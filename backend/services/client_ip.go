package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type clientIPKey struct{}

func remoteIP(remote string) string {
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	if ip, err := netip.ParseAddr(remote); err == nil {
		return ip.Unmap().String()
	}
	return "unknown"
}

// ClientIP accepts forwarded addresses only after the trusted proxy middleware
// has verified the direct peer. Invalid addresses share one limited bucket.
func ClientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPKey{}).(string); ok {
		return ip
	}
	return remoteIP(r.RemoteAddr)
}

// ClientIPMiddleware walks the chain from the trusted direct peer towards the
// visitor and stops at the first untrusted address. No proxy is trusted by default.
func ClientIPMiddleware(raw string) (func(http.Handler) http.Handler, error) {
	var trusted []netip.Prefix
	for _, value := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS entry %q: %w", value, err)
		}
		trusted = append(trusted, prefix)
	}
	isTrusted := func(value string) bool {
		ip, err := netip.ParseAddr(value)
		if err != nil {
			return false
		}
		for _, prefix := range trusted {
			if prefix.Contains(ip) {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := remoteIP(r.RemoteAddr)
			if isTrusted(ip) {
				chain := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
				for i := len(chain) - 1; i >= 0 && isTrusted(ip); i-- {
					candidate, err := netip.ParseAddr(strings.TrimSpace(chain[i]))
					if err != nil {
						ip = remoteIP(r.RemoteAddr)
						break
					}
					ip = candidate.Unmap().String()
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}, nil
}
