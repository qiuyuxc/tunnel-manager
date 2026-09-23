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
	"tunnel-manager/services"
	"tunnel-manager/setup"
	"tunnel-manager/store"
)

// Version is the current application version. The -slim suffix marks the
// single-user edition so the update check only compares against slim releases.
const Version = "v2.7.0-slim"

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
	allowPasswordLogin := flag.Bool("allow-password-login", false, "Re-enable password sign-in for the panel and every account")
	flag.Parse()

	// Storage comes from the environment (containers, scripted installs) or
	// from the document the setup wizard writes; uploads and heartbeat logs
	// stay on the filesystem next to it.
	state, err := setup.Resolve()
	if err != nil {
		log.Fatalf("read storage configuration: %v", err)
	}
	dataDir, err := setup.EnsureDataDir(state.Storage, state.Source)
	if err != nil {
		log.Fatalf("create data directory: %v", err)
	}

	// Handle password reset CLI commands (don't require CF credentials)
	if *resetPassword || *setPassword != "" {
		st := installedStore(state.Storage)
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

	// Recovery path for a panel that switched to passkeys only: clearing the
	// switch needs no working sign-in method, which is exactly the situation
	// this flag exists for.
	if *allowPasswordLogin {
		st := installedStore(state.Storage)
		settings := st.GetAppSettings()
		settings.PasswordLoginDisabled = false
		if err := st.SetAppSettings(settings); err != nil {
			log.Fatalf("re-enable password sign-in: %v", err)
		}
		cleared := 0
		for _, user := range st.ListUsers() {
			if !user.PasswordLoginDisabled {
				continue
			}
			if err := st.SetUserPasswordLoginDisabled(user.ID, false); err != nil {
				log.Fatalf("re-enable password sign-in for %s: %v", user.Username, err)
			}
			cleared++
		}
		fmt.Printf("========================================\n")
		fmt.Printf("  已恢复密码登录\n")
		fmt.Printf("  面板开关：关闭\n")
		fmt.Printf("  账户开关：清除 %d 个\n", cleared)
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
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "frontend/dist"
	}

	if apiToken == "" || accountID == "" {
		log.Printf("Cloudflare static credentials are incomplete; connect through OAuth in global settings")
	}

	// Initialize dependencies. The panel is only built once an administrator
	// exists; before that the process serves the install wizard.
	var panelStore *store.Store
	if state.Source != setup.SourceUnset {
		panelStore = store.NewStore(state.Storage.DSN())
	}
	if panelStore == nil || !panelStore.Installed() {
		serveSetup(state, port, staticDir)
		return
	}
	st := panelStore
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
	heartbeatLog := services.NewHeartbeatLog(filepath.Join(dataDir, "heartbeats.json"))
	monitorRunner := services.NewRunner(st, heartbeatLog)
	labRunner := services.NewLabIPSelectorRunner(st, encryptionKey)
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
	go labRunner.Start(context.Background())
	heartbeatLog.StartFlusher(10 * time.Second)

	// Monitors management
	monitorsHandler := handlers.NewMonitorsHandler(st, heartbeatLog, monitorRunner, domainService)
	uploadsDir := filepath.Join(dataDir, "uploads")
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

	labHandler := handlers.NewLabHandler(st, labRunner, encryptionKey)

	authHandler := handlers.NewAuthHandler(st, encryptionKey)
	managementHandler := handlers.NewManagementHandler(st, encryptionKey)
	passkeyService := services.NewPasskeyService(st)
	passkeyHandler := handlers.NewPasskeyHandler(st, passkeyService, adminHandler)

	notifier := services.NewNotifier(st, encryptionKey)
	adminHandler.SetNotifier(notifier)

	// Sign-in rate limiting. Counters are in memory, so a restart clears every
	// lockout — the panel can always be reached again by the operator who owns
	// the process, which is the property that matters more than persistence.
	throttle := handlers.NewThrottle(st)
	adminHandler.SetThrottle(throttle)
	authHandler.SetThrottle(throttle)
	throttle.SetNotifier(func(userID, username, ip string, retryAfter time.Duration) {
		notifier.NotifyLockout(username, ip, retryAfter)
	})
	notifyHandler := handlers.NewNotifyHandler(st, encryptionKey, notifier)

	mw := &handlers.Middleware{
		APIKey:       apiKey,
		AdminHandler: adminHandler,
		CF:           cf,
		Store:        st,
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	// The panel ships a ~450 KB JS bundle and the generated font stylesheet
	// (116 KB of @font-face rules, ~9 KB gzipped); both are text, so compress.
	r.Use(chimw.Compress(5))
	r.Use(mw.CORS)
	r.Use(handlers.StatusDomainRedirect(st))

	r.Route("/api", func(r chi.Router) {
		// Public site branding
		r.Get("/site", configHandler.GetSiteSettings)

		// Public auth config (Turnstile/site key) and password recovery by email.
		r.Get("/auth/config", authHandler.AuthConfig)
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
		r.Post("/admin/2fa/setup", mw.SessionOnly(adminHandler.SetupTOTP))
		r.Post("/admin/2fa/confirm", mw.SessionOnly(adminHandler.ConfirmTOTP))
		r.Get("/admin/2fa/status", mw.SessionOnly(adminHandler.TOTPStatus))
		r.Post("/admin/2fa/disable", mw.SessionOnly(adminHandler.DisableTOTP))

		// Passkeys: account management is session-only (an API key has no
		// account to bind a credential to), while the sign-in ceremonies are
		// public and complete a login by themselves.
		r.Get("/account/passkeys", mw.SessionOnly(passkeyHandler.Settings))
		r.Post("/account/passkeys/begin", mw.SessionOnly(passkeyHandler.BeginRegistration))
		r.Post("/account/passkeys/finish", mw.SessionOnly(passkeyHandler.FinishRegistration))
		r.Put("/account/passkeys/{id}", mw.SessionOnly(passkeyHandler.Rename))
		r.Delete("/account/passkeys/{id}", mw.SessionOnly(passkeyHandler.Delete))
		r.Put("/account/password-login", mw.SessionOnly(passkeyHandler.UpdatePasswordLogin))
		r.Post("/auth/passkey/login/begin", passkeyHandler.LoginBegin)
		r.Post("/auth/passkey/login/finish", passkeyHandler.LoginFinish)
		r.Post("/admin/login/2fa/passkey/begin", passkeyHandler.TwoFactorBegin)
		r.Post("/admin/login/2fa/passkey/finish", passkeyHandler.TwoFactorFinish)

		// Panel settings (single administrator). Kept from the old admin console:
		// registration/invite/audit management is gone, but these system-config
		// endpoints back features that remain (SMTP alerts, Cloudflare OAuth
		// client, encryption key, passkey/Turnstile/rate-limit hardening).
		r.Route("/admin", func(r chi.Router) {
			r.Put("/users/{id}/password-login", mw.Auth(passkeyHandler.AdminUpdatePasswordLogin))
			r.Get("/settings", mw.Auth(managementHandler.GetAppSettings))
			r.Put("/settings", mw.Auth(managementHandler.UpdateAppSettings))
			r.Get("/smtp", mw.Auth(managementHandler.GetSMTP))
			r.Put("/smtp", mw.Auth(managementHandler.UpdateSMTP))
			r.Post("/smtp/test", mw.Auth(managementHandler.TestSMTP))
			r.Get("/oauth", mw.Auth(managementHandler.GetOAuthConfig))
			r.Put("/oauth", mw.Auth(managementHandler.SaveOAuthConfig))
			r.Get("/encryption-key", mw.Auth(managementHandler.GetEncryptionKeyStatus))
			r.Put("/encryption-key", mw.Auth(managementHandler.SaveEncryptionKey))
		})

		// Cloudflare OAuth endpoints. The callback authenticates through single-use state.
		r.Get("/cloudflare/oauth/status", mw.SessionOnly(cloudflareOAuthHandler.Status))
		r.Post("/cloudflare/oauth/start", mw.Auth(cloudflareOAuthHandler.Start))
		r.Put("/cloudflare/oauth/account", mw.Auth(cloudflareOAuthHandler.SelectAccount))
		r.Delete("/cloudflare/oauth", mw.Auth(cloudflareOAuthHandler.Disconnect))
		r.Put("/cloudflare/oauth/connection", mw.Auth(cloudflareOAuthHandler.ActivateConnection))
		r.Get("/cloudflare/oauth/callback", cloudflareOAuthHandler.Callback)

		// Config endpoints: selections and branding for the administrator.
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

		// DNS record endpoints
		r.Get("/zones/{zoneID}/dns-records", mw.Auth(dnsHandler.List))
		r.Post("/zones/{zoneID}/dns-records", mw.Auth(dnsHandler.Create))
		r.Put("/zones/{zoneID}/dns-records/{recordID}", mw.Auth(dnsHandler.Update))
		r.Delete("/zones/{zoneID}/dns-records/{recordID}", mw.Auth(dnsHandler.Delete))

		// Domain binding endpoints
		r.Post("/domain/bind", mw.Auth(domainHandler.BindDomain))
		r.Post("/domain/bind-batch", mw.Auth(domainHandler.BindDomainsBatch))
		r.Post("/domain/fallback", mw.Auth(domainHandler.SetFallbackOrigin))

		// Service health monitoring
		r.Get("/monitor/services", mw.Auth(monitorHandler.ServiceStatus))

		// Experimental lab (gated by the experimental-features toggle in the UI)
		r.Get("/lab/ip-selector", mw.Auth(labHandler.GetSettings))
		r.Put("/lab/ip-selector", mw.Auth(labHandler.SaveSettings))
		r.Get("/lab/ip-selector/status", mw.Auth(labHandler.GetStatus))
		r.Post("/lab/ip-selector/run", mw.Auth(labHandler.Run))

		// Monitor projects (uptime-style)
		r.Get("/monitors", mw.Auth(monitorsHandler.List))
		r.Post("/monitors", mw.Auth(monitorsHandler.Create))
		r.Get("/monitors/overview", mw.Auth(monitorsHandler.Overview))
		r.Get("/monitors/{monitorID}", mw.Auth(monitorsHandler.Get))
		r.Put("/monitors/{monitorID}", mw.Auth(monitorsHandler.Update))
		r.Delete("/monitors/{monitorID}", mw.Auth(monitorsHandler.Delete))
		r.Get("/monitors/{monitorID}/alerts", mw.Auth(monitorsHandler.AlertLogs))
		r.Post("/monitors/{monitorID}/check", mw.Auth(monitorsHandler.CheckNow))
		r.Post("/monitors/{monitorID}/targets", mw.Auth(monitorsHandler.AddTarget))
		r.Put("/monitors/{monitorID}/targets/{targetID}", mw.Auth(monitorsHandler.EditTarget))
		r.Delete("/monitors/{monitorID}/targets/{targetID}", mw.Auth(monitorsHandler.RemoveTarget))

		// Cursor feed of monitor alerts. Polled by the Android shell so alerts
		// reach the phone without a browser being open.
		r.Get("/alerts", mw.Auth(monitorsHandler.AlertsFeed))

		// Status-page icon uploads
		r.Post("/uploads", mw.Auth(uploadsHandler.UploadImage))

		// Public status page payload (token-scoped, unauthenticated)
		r.Get("/public/status/{token}", monitorsHandler.PublicStatus)

		// Per-user notification preferences: outbound email/Telegram alerts.
		r.Get("/notify/settings", mw.Auth(notifyHandler.GetSettings))
		r.Put("/notify/settings", mw.Auth(notifyHandler.SaveSettings))
		r.Post("/notify/test", mw.Auth(notifyHandler.TestNotify))

		// Health check (no auth)
		r.Get("/health", healthHandler)
	})

	// Serve frontend static files (SPA fallback)
	// Android Digital Asset Links: fetched from the site root, so it has to be
	// registered before the SPA catch-all swallows the path.
	r.Get("/.well-known/assetlinks.json", passkeyHandler.AssetLinks)

	// Uploaded status-page images (before the SPA catch-all)
	r.Get("/uploads/*", uploadsHandler.Serve)

	mountStatic(r, staticDir)

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
