package handlers

import (
	"net"
	"net/http"
	"strings"

	"tunnel-manager/store"
)

// hostWithoutPort normalizes a Host header for comparison against stored
// hostnames: the port is dropped, the trailing root label is trimmed and the
// result is lowercased.
func hostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	if stripped, _, err := net.SplitHostPort(host); err == nil {
		host = stripped
	}
	return strings.ToLower(strings.TrimSuffix(host, "."))
}

// StatusDomainRedirect confines a status page custom domain to that monitor's
// public page and the assets needed to render it. Requests for the main SPA,
// authenticated APIs, and other monitors are deliberately hidden with 404.
func StatusDomainRedirect(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target, published, configured := statusDomainTarget(st, hostWithoutPort(r.Host))
			if !configured {
				next.ServeHTTP(w, r)
				return
			}
			if !published || target == "" || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
				statusDomainNotFound(w, r)
				return
			}
			if r.URL.Path == "/" {
				location := "/status/" + target
				if r.URL.RawQuery != "" {
					location += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, location, http.StatusFound)
				return
			}
			if !statusDomainPublicPath(r.URL.Path, target) {
				statusDomainNotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func statusDomainTarget(st *store.Store, host string) (target string, published, configured bool) {
	if host == "" {
		return "", false, false
	}
	for _, monitor := range st.GetConfig().Monitors {
		if monitor.PublicDomain == "" || !strings.EqualFold(monitor.PublicDomain, host) {
			continue
		}
		target = monitor.PublicSlug
		if target == "" {
			target = monitor.PublicToken
		}
		return target, monitor.PublishEnabled, true
	}
	return "", false, false
}

func statusDomainPublicPath(requestPath, target string) bool {
	if strings.Contains(requestPath, "..") || strings.Contains(requestPath, "\\") {
		return false
	}
	switch requestPath {
	case "/status/" + target,
		"/api/public/status/" + target,
		"/api/site",
		"/icon.webp":
		return true
	default:
		return strings.HasPrefix(requestPath, "/assets/") ||
			strings.HasPrefix(requestPath, "/uploads/")
	}
}

func statusDomainNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	http.NotFound(w, r)
}
