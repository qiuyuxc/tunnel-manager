package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPTrustsOnlyConfiguredProxyChain(t *testing.T) {
	middleware, err := ClientIPMiddleware("10.0.0.0/8,::1/128")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ peer, chain, want string }{
		{"198.51.100.9:80", "203.0.113.1", "198.51.100.9"},
		{"10.0.0.1:80", "203.0.113.1, 198.51.100.9", "198.51.100.9"},
		{"10.0.0.1:80", "203.0.113.1, 10.0.0.2", "203.0.113.1"},
		{"10.0.0.1:80", "junk, 10.0.0.2", "10.0.0.1"},
		{"[::1]:80", "::ffff:198.51.100.9", "198.51.100.9"},
		{"invalid-peer", "203.0.113.1", "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.peer+tc.chain, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Forwarded-For", tc.chain)
			r.Header.Set("CF-Connecting-IP", "203.0.113.55")
			middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if got := ClientIP(req); got != tc.want {
					t.Fatalf("got %q, want %q", got, tc.want)
				}
			})).ServeHTTP(httptest.NewRecorder(), r)
		})
	}
	if _, err := ClientIPMiddleware("trust-anyone"); err == nil {
		t.Fatal("accepted invalid trust configuration")
	}
}
