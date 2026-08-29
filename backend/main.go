package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// Version is the current application version.
const Version = "v2.2.0-test.1"

func main() {
	// Pin the process timezone to Asia/Shanghai so every user-facing time
	// (login/notification timestamps, bot status, logs) is shown in the
	// panel's local time instead of UTC.
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		time.Local = loc
	} else {
		time.Local = time.FixedZone("UTC+8", 8*3600)
	}
	loadDotEnv(".env", "../.env")

	// CLI flags for password management
	resetPassword := flag.Bool("reset-password", false, "Generate a new random admin password")
	setPassword := flag.String("set-password", "", "Set admin password to a specific value")
	flag.Parse()

	storePath := os.Getenv("STORE_PATH")
	if storePath == "" {
		storePath = "data/tunnel-manager.db"
	}

	// Handle password reset CLI commands (don't require CF credentials)
	if *resetPassword || *setPassword != "" {
		st := store.NewStore(storePath)
		username, _ := st.GetAdminCredentials()

		var newPassword string
		if *setPassword != "" {
			newPassword = *setPassword
		} else {
			newPassword = "admin123"
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if apiToken == "" || accountID == "" {
		log.Printf("Cloudflare static credentials are incomplete; connect through OAuth in global settings")
	}

	// Initialize dependencies
	st := store.NewStore(storePath)
	var encryptionKey []byte
	encryptionKey = resolveEncryptionKey(st, os.Getenv("APP_ENCRYPTION_KEY"))
	cf := services.NewCloudflareClient(apiToken, accountID)
	cf.SetSessionStore(st)
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
	monitorRunner.SetMailer(func() *services.Mailer {
		settings := st.GetSMTPSettings()
		if !settings.Configured() || settings.Password == "" {
			return nil
		}
		plain, err := auth.DecryptSecret(encryptionKey, "smtp-password", settings.Password)
		if err != nil {
			log.Printf("decrypt SMTP password: %v", err)
			return nil
		}
		return services.NewMailer(settings, string(plain))
	})
	go monitorRunner.Start(context.Background())
	heartbeatLog.StartFlusher(10 * time.Second)

	// Monitors management
	monitorsHandler := handlers.NewMonitorsHandler(st, heartbeatLog, monitorRunner, domainService)
	uploadsDir := filepath.Join(filepath.Dir(storePath), "uploads")
	uploadsHandler := handlers.NewUploadsHandler(uploadsDir)
	uploadsHandler.SetStore(st)

	// Initialize handlers
	configHandler := handlers.NewConfigHandler(st)
	tunnelHandler := handlers.NewTunnelHandler(cf, st)
	domainHandler := handlers.NewDomainHandler(domainService)
	dnsHandler := handlers.NewDNSHandler(cf)
	monitorHandler := handlers.NewMonitorHandler(cf, st)
	adminHandler := handlers.NewAdminHandler(st, encryptionKey)
	cloudflareOAuthHandler := handlers.NewCloudflareOAuthHandler(st, cloudflareOAuth, cf, adminHandler)

	telegramBot := services.NewTelegramBot(st, cf, domainService)
	userTelegramManager := services.NewUserTelegramManager(st, cf, domainService, encryptionKey)
	telegramHandler := handlers.NewTelegramHandler(st, telegramBot, userTelegramManager, encryptionKey)

	authHandler := handlers.NewAuthHandler(st, encryptionKey)
	managementHandler := handlers.NewManagementHandler(st, encryptionKey)

	notifier := services.NewNotifier(st, encryptionKey)
	adminHandler.SetNotifier(notifier)
	notifyHandler := handlers.NewNotifyHandler(st, encryptionKey, notifier)

	mw := &handlers.Middleware{
		APIKey:       apiKey,
		AdminHandler: adminHandler,
		CF:           cf,
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.CORS)
	r.Use(handlers.StatusDomainRedirect(st))

	r.Route("/api", func(r chi.Router) {
		// Public site branding
		r.Get("/site", configHandler.GetSiteSettings)

		// Registration flows (public; policy enforced server-side)
		r.Get("/auth/config", authHandler.AuthConfig)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/send-code", authHandler.SendCode)
		r.Post("/auth/forgot-password", authHandler.ForgotPassword)
		r.Post("/auth/reset-password", authHandler.ResetPassword)
		r.Get("/auth/me", mw.Auth(authHandler.Me))

		// Login endpoints (no auth required)
		r.Post("/admin/login", adminHandler.Login)
		r.Post("/admin/login/2fa", adminHandler.LoginTwoFactor)
		r.Post("/admin/logout", adminHandler.Logout)
		r.Get("/admin/status", adminHandler.Status)

		// Account management (any authenticated user)
		r.Put("/admin/profile", mw.Auth(adminHandler.ChangeProfile))
		r.Post("/account/avatar", mw.Auth(uploadsHandler.UploadAvatar))
		r.Put("/admin/password", mw.Auth(adminHandler.ChangePassword))
		r.Put("/admin/username", mw.Auth(adminHandler.ChangeUsername))
		r.Put("/admin/email", mw.Auth(adminHandler.ChangeEmail))
		r.Put("/admin/email", mw.Auth(adminHandler.ChangeEmail))
		r.Post("/admin/2fa/setup", mw.SessionOnly(adminHandler.SetupTOTP))
		r.Post("/admin/2fa/confirm", mw.SessionOnly(adminHandler.ConfirmTOTP))
		r.Get("/admin/2fa/status", mw.SessionOnly(adminHandler.TOTPStatus))
		r.Post("/admin/2fa/disable", mw.SessionOnly(adminHandler.DisableTOTP))

		// Admin backend (administrator only)
		adminOnly := func(next http.HandlerFunc) http.HandlerFunc {
			return mw.Auth(mw.RequireAdmin(next))
		}
		r.Route("/admin", func(r chi.Router) {
			r.Get("/users", adminOnly(managementHandler.ListUsers))
			r.Post("/users", adminOnly(managementHandler.CreateUser))
			r.Put("/users/{id}/status", adminOnly(managementHandler.UpdateUserStatus))
			r.Put("/users/{id}/group", adminOnly(managementHandler.UpdateUserGroup))
			r.Put("/users/{id}/password", adminOnly(managementHandler.ResetUserPassword))
			r.Delete("/users/{id}", adminOnly(managementHandler.DeleteUser))
			r.Get("/groups", adminOnly(managementHandler.ListGroups))
			r.Post("/groups", adminOnly(managementHandler.CreateGroup))
			r.Put("/groups/{id}", adminOnly(managementHandler.UpdateGroup))
			r.Delete("/groups/{id}", adminOnly(managementHandler.DeleteGroup))
			r.Get("/invites", adminOnly(managementHandler.ListInvites))
			r.Post("/invites", adminOnly(managementHandler.CreateInvite))
			r.Put("/invites/{code}", adminOnly(managementHandler.UpdateInvite))
			r.Delete("/invites/{code}", adminOnly(managementHandler.DeleteInvite))
			r.Get("/settings", adminOnly(managementHandler.GetAppSettings))
			r.Put("/settings", adminOnly(managementHandler.UpdateAppSettings))
			r.Get("/smtp", adminOnly(managementHandler.GetSMTP))
			r.Put("/smtp", adminOnly(managementHandler.UpdateSMTP))
			r.Post("/smtp/test", adminOnly(managementHandler.TestSMTP))
			r.Get("/oauth", adminOnly(managementHandler.GetOAuthConfig))
			r.Put("/oauth", adminOnly(managementHandler.SaveOAuthConfig))
			r.Get("/encryption-key", adminOnly(managementHandler.GetEncryptionKeyStatus))
			r.Put("/encryption-key", adminOnly(managementHandler.SaveEncryptionKey))
		})

		// Cloudflare OAuth endpoints. The callback authenticates through single-use state.
		r.Get("/cloudflare/oauth/status", mw.SessionOnly(cloudflareOAuthHandler.Status))
		r.Post("/cloudflare/oauth/start", mw.Auth(mw.RequirePerm(models.PermOAuthConnect, cloudflareOAuthHandler.Start)))
		r.Put("/cloudflare/oauth/account", mw.Auth(mw.RequirePerm(models.PermOAuthConnect, cloudflareOAuthHandler.SelectAccount)))
		r.Delete("/cloudflare/oauth", mw.Auth(mw.RequirePerm(models.PermOAuthConnect, cloudflareOAuthHandler.Disconnect)))
		r.Put("/cloudflare/oauth/connection", mw.Auth(mw.RequirePerm(models.PermOAuthConnect, cloudflareOAuthHandler.ActivateConnection)))
		r.Get("/cloudflare/oauth/callback", cloudflareOAuthHandler.Callback)

		// Config endpoints: user selections for everyone, branding for admins
		r.Get("/config", mw.Auth(configHandler.GetConfig))
		r.Post("/config/tunnel", mw.Auth(configHandler.SetTunnelSelection))
		r.Post("/config/service", mw.Auth(configHandler.SetServiceURL))
		r.Post("/config/preferred-cname", mw.Auth(mw.RequireAdmin(configHandler.SetPreferredCNAME)))
		r.Put("/config/site", mw.Auth(mw.RequireAdmin(configHandler.SetSiteSettings)))
		r.Put("/config/cname-presets", mw.Auth(mw.RequireAdmin(configHandler.SetCNAMEPresets)))

		// Tunnel endpoints
		r.Get("/tunnels", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.ListTunnels)))
		r.Post("/tunnels", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.CreateTunnel)))
		r.Get("/tunnels/{tunnelID}", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.GetTunnelDetail)))
		r.Delete("/tunnels/{tunnelID}", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.DeleteTunnel)))
		r.Post("/tunnels/{tunnelID}/ingress", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.AddIngressRule)))
		r.Put("/tunnels/{tunnelID}/ingress", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.UpdateIngressRule)))
		r.Delete("/tunnels/{tunnelID}/ingress", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.DeleteIngressRule)))
		r.Get("/zones", mw.Auth(mw.RequirePerm(models.PermTunnels, tunnelHandler.ListZones)))

		// DNS record endpoints
		r.Get("/zones/{zoneID}/dns-records", mw.Auth(mw.RequirePerm(models.PermDNS, dnsHandler.List)))
		r.Post("/zones/{zoneID}/dns-records", mw.Auth(mw.RequirePerm(models.PermDNS, dnsHandler.Create)))
		r.Put("/zones/{zoneID}/dns-records/{recordID}", mw.Auth(mw.RequirePerm(models.PermDNS, dnsHandler.Update)))
		r.Delete("/zones/{zoneID}/dns-records/{recordID}", mw.Auth(mw.RequirePerm(models.PermDNS, dnsHandler.Delete)))

		// Domain binding endpoints
		r.Post("/domain/bind", mw.Auth(mw.RequirePerm(models.PermDomainBind, domainHandler.BindDomain)))
		r.Post("/domain/bind-batch", mw.Auth(mw.RequirePerm(models.PermDomainBind, domainHandler.BindDomainsBatch)))
		r.Post("/domain/fallback", mw.Auth(mw.RequirePerm(models.PermDomainBind, domainHandler.SetFallbackOrigin)))

		// Service health monitoring
		r.Get("/monitor/services", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorHandler.ServiceStatus)))

		// Monitor projects (uptime-style)
		r.Get("/monitors", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.List)))
		r.Post("/monitors", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.Create)))
		r.Get("/monitors/overview", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.Overview)))
		r.Get("/monitors/{monitorID}", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.Get)))
		r.Put("/monitors/{monitorID}", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.Update)))
		r.Delete("/monitors/{monitorID}", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.Delete)))
		r.Get("/monitors/{monitorID}/alerts", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.AlertLogs)))
		r.Post("/monitors/{monitorID}/check", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.CheckNow)))
		r.Post("/monitors/{monitorID}/targets", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.AddTarget)))
		r.Put("/monitors/{monitorID}/targets/{targetID}", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.EditTarget)))
		r.Delete("/monitors/{monitorID}/targets/{targetID}", mw.Auth(mw.RequirePerm(models.PermMonitors, monitorsHandler.RemoveTarget)))

		// Status-page icon uploads
		r.Post("/uploads", mw.Auth(mw.RequirePerm(models.PermMonitors, uploadsHandler.UploadImage)))

		// Public status page payload (token-scoped, unauthenticated)
		r.Get("/public/status/{token}", monitorsHandler.PublicStatus)

		// Per-user Telegram remote-control bot (any authenticated user; each
		// account owns an isolated bot)
		r.Get("/telegram/settings", mw.Auth(telegramHandler.GetSettings))
		r.Put("/telegram/settings", mw.Auth(telegramHandler.SaveSettings))
		r.Get("/telegram/status", mw.Auth(telegramHandler.GetStatus))
		r.Post("/telegram/test", mw.Auth(telegramHandler.SendTest))
		r.Post("/telegram/reuse", mw.Auth(telegramHandler.ReuseFromNotify))
		r.Put("/telegram/endpoint", mw.Auth(mw.RequireAdmin(telegramHandler.SaveAPIEndpoint)))
		r.Post("/telegram/webhook", telegramHandler.Webhook) // no auth: verified via secret token

		// Per-user notification preferences (any authenticated user)
		r.Get("/notify/settings", mw.Auth(notifyHandler.GetSettings))
		r.Put("/notify/settings", mw.Auth(func(w http.ResponseWriter, req *http.Request) {
			notifyHandler.SaveSettings(w, req)
			userTelegramManager.Reconcile() // token may change via notifications
		}))
		r.Post("/notify/reuse", mw.Auth(notifyHandler.ReuseFromTelegram))
		r.Post("/notify/test", mw.Auth(notifyHandler.TestNotify))

		// Health check (no auth)
		r.Get("/health", healthHandler)
	})

	// Migrate the legacy global bot into the administrator's per-user
	// preferences, then start one isolated bot per enabled account.
	userTelegramManager.MigrateLegacyAdminBot()
	userTelegramManager.Reconcile()

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

// healthHandler serves the unauthenticated liveness endpoint.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"status\":\"ok\",\"version\":%q}", Version)
}

// loadDotEnv reads KEY=VALUE pairs from the given .env files (variables
// already set in the environment win) so binary deployments pick up the same
// configuration as docker-compose. Missing files are ignored.
func loadDotEnv(paths ...string) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), "\"'")
			if key == "" {
				continue
			}
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
		log.Printf("loaded environment overrides from %s", path)
		return
	}
}

// resolveEncryptionKey resolves the application encryption key: the
// APP_ENCRYPTION_KEY environment variable wins, then the key stored in the
// database (settable from the admin console), otherwise one is generated and
// persisted so a restart keeps it stable.
func resolveEncryptionKey(st *store.Store, envKey string) []byte {
	if envKey != "" {
		key, err := auth.ParseEncryptionKey(envKey)
		if err != nil {
			log.Fatalf("invalid APP_ENCRYPTION_KEY: %v", err)
		}
		return key
	}
	if stored := st.GetEncryptionKeyRaw(); stored != "" {
		key, err := auth.ParseEncryptionKey(stored)
		if err == nil {
			return key
		}
		log.Printf("stored encryption key is invalid, generating a new one: %v", err)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		log.Fatalf("generate encryption key: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)
	if err := st.SetEncryptionKeyRaw(encoded); err != nil {
		log.Fatalf("store generated encryption key: %v", err)
	}
	log.Printf("已自动生成应用加密密钥并保存在数据库中；更换密钥会使已保存的授权与密文失效")
	key, err := auth.ParseEncryptionKey(encoded)
	if err != nil {
		log.Fatalf("parse generated encryption key: %v", err)
	}
	return key
}
