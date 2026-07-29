package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"tunnel-manager/auth"
	"tunnel-manager/handlers"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

func main() {
	// CLI flags for password management
	resetPassword := flag.Bool("reset-password", false, "Generate a new random admin password")
	setPassword := flag.String("set-password", "", "Set admin password to a specific value")
	flag.Parse()

	storePath := os.Getenv("STORE_PATH")
	if storePath == "" {
		storePath = "data/config.json"
	}

	// Handle password reset CLI commands (don't require CF credentials)
	if *resetPassword || *setPassword != "" {
		st := store.NewStore(storePath)
		username, _ := st.GetAdminCredentials()

		var newPassword string
		if *setPassword != "" {
			newPassword = *setPassword
		} else {
			newPassword = generateRandomPassword(12)
		}

		if err := st.SetAdminCredentials(username, store.HashPassword(newPassword)); err != nil {
			log.Fatalf("reset administrator password: %v", err)
		}
		fmt.Printf("========================================\n")
		fmt.Printf("  密码已重置\n")
		fmt.Printf("  用户名: %s\n", username)
		fmt.Printf("  新密码: %s\n", newPassword)
		fmt.Printf("  请登录后立即修改密码！\n")
		fmt.Printf("========================================\n")
		return
	}

	apiToken := os.Getenv("CF_API_TOKEN")
	accountID := os.Getenv("CF_ACCOUNT_ID")
	apiKey := os.Getenv("API_KEY")
	var encryptionKey []byte
	if encodedKey := os.Getenv("APP_ENCRYPTION_KEY"); encodedKey != "" {
		var err error
		encryptionKey, err = auth.ParseEncryptionKey(encodedKey)
		if err != nil {
			log.Fatalf("invalid APP_ENCRYPTION_KEY: %v", err)
		}
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if apiToken == "" || accountID == "" {
		log.Fatal("CF_API_TOKEN and CF_ACCOUNT_ID environment variables are required")
	}

	// Initialize dependencies
	st := store.NewStore(storePath)
	cf := services.NewCloudflareClient(apiToken, accountID)

	// Initialize services
	domainService := services.NewDomainService(cf, st)

	// Initialize handlers
	configHandler := handlers.NewConfigHandler(st)
	tunnelHandler := handlers.NewTunnelHandler(cf)
	domainHandler := handlers.NewDomainHandler(domainService)
	adminHandler := handlers.NewAdminHandler(st, encryptionKey)

	telegramBot := services.NewTelegramBot(st, cf, domainService)
	telegramHandler := handlers.NewTelegramHandler(st, telegramBot)

	mw := &handlers.Middleware{
		APIKey:       apiKey,
		AdminHandler: adminHandler,
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.CORS)

	r.Route("/api", func(r chi.Router) {
		// Public site branding
		r.Get("/site", configHandler.GetSiteSettings)

		// Admin endpoints (no auth required)
		r.Post("/admin/login", adminHandler.Login)
		r.Post("/admin/login/2fa", adminHandler.LoginTwoFactor)
		r.Post("/admin/logout", adminHandler.Logout)
		r.Get("/admin/status", adminHandler.Status)

		// Account management preserves existing Auth behavior; 2FA management requires a real session.
		r.Put("/admin/password", mw.Auth(adminHandler.ChangePassword))
		r.Put("/admin/username", mw.Auth(adminHandler.ChangeUsername))
		r.Post("/admin/2fa/setup", mw.SessionOnly(adminHandler.SetupTOTP))
		r.Post("/admin/2fa/confirm", mw.SessionOnly(adminHandler.ConfirmTOTP))
		r.Get("/admin/2fa/status", mw.SessionOnly(adminHandler.TOTPStatus))
		r.Post("/admin/2fa/disable", mw.SessionOnly(adminHandler.DisableTOTP))

		// Config endpoints
		r.Get("/config", mw.Auth(configHandler.GetConfig))
		r.Post("/config/tunnel", mw.Auth(configHandler.SetTunnelSelection))
		r.Post("/config/service", mw.Auth(configHandler.SetServiceURL))
		r.Post("/config/preferred-cname", mw.Auth(configHandler.SetPreferredCNAME))
		r.Put("/config/site", mw.Auth(configHandler.SetSiteSettings))
		r.Put("/config/cname-presets", mw.Auth(configHandler.SetCNAMEPresets))

		// Tunnel endpoints
		r.Get("/tunnels", mw.Auth(tunnelHandler.ListTunnels))
		r.Get("/tunnels/{tunnelID}", mw.Auth(tunnelHandler.GetTunnelDetail))
		r.Post("/tunnels/{tunnelID}/ingress", mw.Auth(tunnelHandler.AddIngressRule))
		r.Put("/tunnels/{tunnelID}/ingress", mw.Auth(tunnelHandler.UpdateIngressRule))
		r.Get("/zones", mw.Auth(tunnelHandler.ListZones))

		// Domain endpoints
		r.Post("/domain/bind", mw.Auth(domainHandler.BindDomain))
		r.Post("/domain/bind-batch", mw.Auth(domainHandler.BindDomainsBatch))
		r.Post("/domain/fallback", mw.Auth(domainHandler.SetFallbackOrigin))

		// Telegram bot endpoints
		r.Get("/telegram/settings", mw.Auth(telegramHandler.GetSettings))
		r.Put("/telegram/settings", mw.Auth(telegramHandler.SaveSettings))
		r.Get("/telegram/status", mw.Auth(telegramHandler.GetStatus))
		r.Post("/telegram/test", mw.Auth(telegramHandler.SendTest))
		r.Post("/telegram/webhook", telegramHandler.Webhook) // no auth: verified via secret token

		// Health check (no auth)
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})
	})

	// Auto-start Telegram bot if enabled
	if st.GetConfig().TGBotEnabled {
		if err := telegramBot.Start(); err != nil {
			log.Printf("telegram bot auto-start failed: %v", err)
		}
	}

	// Serve frontend static files (SPA fallback)
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "frontend/dist"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			// Try to serve the file directly
			path := filepath.Join(staticDir, req.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) || strings.HasSuffix(req.URL.Path, "/") {
				// SPA fallback: serve index.html for missing routes
				http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, req)
		})
		log.Printf("Serving static files from %s", staticDir)
	}

	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func generateRandomPassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
}
