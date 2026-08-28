package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

// Middleware holds shared dependencies for handlers
type Middleware struct {
	APIKey       string
	AdminHandler *AdminHandler
	CF           *services.CloudflareClient
}

// contextKey carries the authenticated identity through a request.
type contextKey struct{}

// cfContextKey carries the per-user Cloudflare client.
type cfContextKey struct{}

// UserCF returns the Cloudflare client bound to the requesting account, or
// nil on unauthenticated requests.
func UserCF(r *http.Request) *services.CloudflareClient {
	if cf, ok := r.Context().Value(cfContextKey{}).(*services.CloudflareClient); ok {
		return cf
	}
	return nil
}

// sessionUID returns the authenticated account id, or "" when absent.
func sessionUID(r *http.Request) string {
	if user := SessionUser(r); user != nil {
		return user.ID
	}
	return ""
}

// withCF attaches the per-user Cloudflare client to the request.
func withCF(r *http.Request, cf *services.CloudflareClient) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), cfContextKey{}, cf))
}

// CORS wraps a handler with CORS headers
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Auth-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// sessionFromToken resolves the account behind a session token.
func (m *Middleware) sessionFromToken(token string) (models.SessionUser, bool) {
	if token == "" || m.AdminHandler == nil {
		return models.SessionUser{}, false
	}
	return m.AdminHandler.ValidateSession(token)
}

// withUser attaches the identity to the request context.
func withUser(r *http.Request, user models.SessionUser) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKey{}, user))
}

// SessionUser returns the authenticated identity attached by the auth
// middleware, or nil when the request is unauthenticated.
func SessionUser(r *http.Request) *models.SessionUser {
	user, _ := r.Context().Value(contextKey{}).(models.SessionUser)
	if user.ID == "" && user.Role == "" {
		return nil
	}
	return &user
}

// Auth wraps a handler with session token or admin API key authentication and
// attaches the resolved identity to the request context.
func (m *Middleware) Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if su, ok := m.sessionFromToken(r.Header.Get("X-Auth-Token")); ok {
			next(w, withCF(withUser(r, su), m.cfFor(su)))
			return
		}

		// Static API key access keeps full administrator reach.
		if m.APIKey != "" {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.URL.Query().Get("api_key")
			}
			if key == m.APIKey {
				su := models.SessionUser{
					Username:    "api-key",
					Role:        models.RoleAdmin,
					Permissions: append([]string(nil), models.AllPermissions...),
				}
				next(w, withCF(withUser(r, su), m.cfFor(su)))
				return
			}
		}

		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}
}

// cfFor derives the account-scoped client for an identity.
func (m *Middleware) cfFor(user models.SessionUser) *services.CloudflareClient {
	if m.CF == nil {
		return nil
	}
	return m.CF.ForUser(user.ID)
}

// SessionOnly wraps security-sensitive account management endpoints and
// accepts only a valid session token.
func (m *Middleware) SessionOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if su, ok := m.sessionFromToken(r.Header.Get("X-Auth-Token")); ok {
			next(w, withUser(r, su))
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
}

// RequirePerm gates a handler behind one group permission. Administrators
// always pass; it must be wrapped by Auth so the identity is present.
func (m *Middleware) RequirePerm(perm string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := SessionUser(r)
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if !user.HasPerm(perm) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "permission denied"})
			return
		}
		next(w, r)
	}
}

// RequireAdmin gates a handler behind the administrator role. It must be
// wrapped by Auth so the identity is present.
func (m *Middleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := SessionUser(r)
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if !user.IsAdmin() {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "administrator access required"})
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func readJSON(r *http.Request, target interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func getBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
