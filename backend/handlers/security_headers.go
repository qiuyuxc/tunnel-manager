package handlers

import (
	"log"
	"net/http"
	"strings"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// SecurityHeaders keeps API responses and uploaded assets from being treated
// as executable HTML and avoids leaking share tokens to external image hosts.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// SafeLogger retains route/status/timing diagnostics while keeping API keys,
// authorization codes and status share tokens out of the access log.
func SafeLogger(next http.Handler) http.Handler {
	return chimw.RequestLogger(&redactedLogFormatter{
		base: &chimw.DefaultLogFormatter{Logger: log.Default(), NoColor: true},
	})(next)
}

type redactedLogFormatter struct {
	base *chimw.DefaultLogFormatter
}

func (f *redactedLogFormatter) NewLogEntry(r *http.Request) chimw.LogEntry {
	safe := r.Clone(r.Context())
	safe.RequestURI = r.URL.EscapedPath()
	for _, prefix := range []string{"/status/", "/api/public/status/"} {
		if strings.HasPrefix(r.URL.Path, prefix) {
			safe.RequestURI = prefix + "[redacted]"
			break
		}
	}
	return f.base.NewLogEntry(safe)
}
