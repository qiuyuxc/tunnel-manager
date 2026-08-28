package handlers

import (
	"net"
	"net/http"
	"strings"

	"tunnel-manager/auth"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// turnstileSecretPurpose names the encrypted Turnstile secret in the store.
const turnstileSecretPurpose = "turnstile-secret"

// verifyTurnstile gates a request when the administrator enabled Cloudflare
// Turnstile. It returns (ok, errorMessage); when disabled it always passes.
// The token is redeemed once via the canonical siteverify endpoint, and any
// verifier failure fails closed.
func verifyTurnstile(st *store.Store, encryptionKey []byte, r *http.Request, token, action string) (bool, string) {
	settings := st.GetAppSettings()
	if !settings.TurnstileEnabled {
		return true, ""
	}
	if settings.TurnstileSecret == "" {
		return false, "人机验证尚未配置，请联系管理员"
	}
	plain, err := auth.DecryptSecret(encryptionKey, turnstileSecretPurpose, settings.TurnstileSecret)
	if err != nil {
		return false, "人机验证服务配置错误，请联系管理员"
	}
	ok, err := services.VerifyTurnstile(string(plain), strings.TrimSpace(token), clientIP(r), action)
	if err != nil {
		return false, "人机验证服务暂时不可用，请稍后重试"
	}
	if !ok {
		return false, "人机验证未通过，请完成验证后重试"
	}
	return true, ""
}

// clientIP extracts the best available client address for siteverify.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
