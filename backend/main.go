package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"tunnel-manager/auth"
	"tunnel-manager/handlers"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// Version is the current application version.
const Version = "v1.16.1"

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
	oauthClientID := os.Getenv("CF_OAUTH_CLIENT_ID")
	oauthClientSecret := os.Getenv("CF_OAUTH_CLIENT_SECRET")
	oauthRedirectURI := os.Getenv("CF_OAUTH_REDIRECT_URI")
	oauthScopes := os.Getenv("CF_OAUTH_SCOPES")
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
		log.Printf("Cloudflare static credentials are incomplete; connect through OAuth in global settings")
	}

	// Initialize dependencies
	st := store.NewStore(storePath)
	cf := services.NewCloudflareClient(apiToken, accountID)
	cloudflareOAuth := services.NewCloudflareOAuth(st, encryptionKey, services.CloudflareOAuthConfig{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		RedirectURI:  oauthRedirectURI,
		Scopes:       oauthScopes,
	})
	cf.SetOAuth(cloudflareOAuth)

	// Initialize services
	domainService := services.NewDomainService(cf, st)

	// Service monitoring heartbeat storage and scheduler
	heartbeatLog := services.NewHeartbeatLog(filepath.Join(filepath.Dir(storePath), "heartbeats.json"))
	monitorRunner := services.NewRunner(st, heartbeatLog)
	go monitorRunner.Start(context.Background())
	heartbeatLog.StartFlusher(10 * time.Second)

	// Monitors management
	monitorsHandler := handlers.NewMonitorsHandler(st, heartbeatLog, monitorRunner)
	uploadsDir := filepath.Join(filepath.Dir(storePath), "uploads")
	uploadsHandler := handlers.NewUploadsHandler(uploadsDir)

	// Initialize handlers
	configHandler := handlers.NewConfigHandler(st)
	tunnelHandler := handlers.NewTunnelHandler(cf, st)
	domainHandler := handlers.NewDomainHandler(domainService)
	dnsHandler := handlers.NewDNSHandler(cf)
	monitorHandler := handlers.NewMonitorHandler(cf, st)
	adminHandler := handlers.NewAdminHandler(st, encryptionKey)
	cloudflareOAuthHandler := handlers.NewCloudflareOAuthHandler(st, cloudflareOAuth, cf, adminHandler)

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

		// Cloudflare OAuth endpoints. The callback authenticates through single-use state.
		r.Get("/cloudflare/oauth/status", mw.SessionOnly(cloudflareOAuthHandler.Status))
		r.Post("/cloudflare/oauth/start", mw.SessionOnly(cloudflareOAuthHandler.Start))
		r.Put("/cloudflare/oauth/account", mw.SessionOnly(cloudflareOAuthHandler.SelectAccount))
		r.Delete("/cloudflare/oauth", mw.SessionOnly(cloudflareOAuthHandler.Disconnect))
		r.Get("/cloudflare/oauth/callback", cloudflareOAuthHandler.Callback)

		// Config endpoints
		r.Get("/config", mw.Auth(configHandler.GetConfig))
		r.Post("/config/tunnel", mw.Auth(configHandler.SetTunnelSelection))
		r.Post("/config/service", mw.Auth(configHandler.SetServiceURL))
		r.Post("/config/preferred-cname", mw.Auth(configHandler.SetPreferredCNAME))
		r.Put("/config/site", mw.Auth(configHandler.SetSiteSettings))
		r.Put("/config/cname-presets", mw.Auth(configHandler.SetCNAMEPresets))

		// Tunnel endpoints
		r.Get("/tunnels", mw.Auth(tunnelHandler.ListTunnels))
		r.Post("/tunnels", mw.Auth(tunnelHandler.CreateTunnel))
		r.Get("/tunnels/{tunnelID}", mw.Auth(tunnelHandler.GetTunnelDetail))
		r.Delete("/tunnels/{tunnelID}", mw.Auth(tunnelHandler.DeleteTunnel))
		r.Post("/tunnels/{tunnelID}/ingress", mw.Auth(tunnelHandler.AddIngressRule))
		r.Put("/tunnels/{tunnelID}/ingress", mw.Auth(tunnelHandler.UpdateIngressRule))
		r.Delete("/tunnels/{tunnelID}/ingress", mw.Auth(tunnelHandler.DeleteIngressRule))
		r.Get("/zones", mw.Auth(tunnelHandler.ListZones))
		r.Get("/zones/{zoneID}/dns-records", mw.Auth(dnsHandler.List))
		r.Post("/zones/{zoneID}/dns-records", mw.Auth(dnsHandler.Create))
		r.Put("/zones/{zoneID}/dns-records/{recordID}", mw.Auth(dnsHandler.Update))
		r.Delete("/zones/{zoneID}/dns-records/{recordID}", mw.Auth(dnsHandler.Delete))

		// Service health monitoring
		r.Get("/monitor/services", mw.Auth(monitorHandler.ServiceStatus))

		// Monitor projects (uptime-style)
		r.Get("/monitors", mw.Auth(monitorsHandler.List))
		r.Post("/monitors", mw.Auth(monitorsHandler.Create))
		r.Get("/monitors/overview", mw.Auth(monitorsHandler.Overview))
		r.Get("/monitors/{monitorID}", mw.Auth(monitorsHandler.Get))
		r.Put("/monitors/{monitorID}", mw.Auth(monitorsHandler.Update))
		r.Delete("/monitors/{monitorID}", mw.Auth(monitorsHandler.Delete))
		r.Post("/monitors/{monitorID}/check", mw.Auth(monitorsHandler.CheckNow))
		r.Post("/monitors/{monitorID}/targets", mw.Auth(monitorsHandler.AddTarget))
		r.Put("/monitors/{monitorID}/targets/{targetID}", mw.Auth(monitorsHandler.EditTarget))
		r.Delete("/monitors/{monitorID}/targets/{targetID}", mw.Auth(monitorsHandler.RemoveTarget))

		// Status-page icon uploads
		r.Post("/uploads", mw.Auth(uploadsHandler.UploadImage))

		// Public status page payload (token-scoped, unauthenticated)
		r.Get("/public/status/{token}", monitorsHandler.PublicStatus)

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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"ok","version":%q}`, Version)
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
	// Uploaded status-page images (before the SPA catch-all)
	r.Get("/uploads/*", uploadsHandler.Serve)

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
