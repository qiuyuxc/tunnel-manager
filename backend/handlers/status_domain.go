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

// StatusDomainRedirect sends visitors arriving on a status page's custom domain
// from the site root to that page. Only the root path is claimed, so every
// other path (the SPA, /api, other status pages) still works on the same
// domain. It runs as middleware, ahead of routing, so it applies whether or not
// the static frontend is being served.
func StatusDomainRedirect(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				next.ServeHTTP(w, r)
				return
			}
			monitor, ok := st.FindMonitorByDomain(hostWithoutPort(r.Host))
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			target := monitor.PublicSlug
			if target == "" {
				target = monitor.PublicToken
			}
			if target == "" {
				next.ServeHTTP(w, r)
				return
			}
			location := "/status/" + target
			if r.URL.RawQuery != "" {
				location += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, location, http.StatusFound)
		})
	}
}
