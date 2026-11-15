package handlers

import (
	"bytes"
	"encoding/base64"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func imageUploadRequest(t *testing.T, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := form.WriteField("filename", filename); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/uploads", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
}

func TestImageUploadRejectsExecutableAndMismatchedContent(t *testing.T) {
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aL1QAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body []byte
		want int
	}{
		{"photo.png", png, http.StatusOK},
		{"photo.jpeg", png, http.StatusBadRequest},
		{"photo.png", []byte("<!DOCTYPE html><script>localStorage.getItem('auth_token')</script>"), http.StatusBadRequest},
		{"photo.webp", []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"), http.StatusBadRequest},
	} {
		h := NewUploadsHandler(t.TempDir())
		w := httptest.NewRecorder()
		h.UploadImage(w, imageUploadRequest(t, tc.name, tc.body))
		if w.Code != tc.want {
			t.Fatalf("%s: got %d: %s", tc.name, w.Code, w.Body.String())
		}
		files, err := os.ReadDir(h.Dir)
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != http.StatusOK && len(files) != 0 {
			t.Fatal("rejected content persisted")
		}
	}
}

func TestLegacyUploadedHTMLCannotBeServedAsExecutableContent(t *testing.T) {
	h := NewUploadsHandler(t.TempDir())
	for _, name := range []string{"old.png", "old.html"} {
		if err := os.WriteFile(filepath.Join(h.Dir, name), []byte("<!DOCTYPE html><script>alert(1)</script>"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	router := chi.NewRouter()
	router.Get("/uploads/*", h.Serve)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/old.png", nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("legacy upload can be sniffed: %d %v", w.Code, w.Header())
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/old.html", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("executable extension served: %d", w.Code)
	}
}

func TestSecurityHeadersAndAuthenticatedCachePolicy(t *testing.T) {
	admin := newTestAdminHandler(t)
	mw := &Middleware{AdminHandler: admin}
	for _, authenticated := range []func(http.HandlerFunc) http.HandlerFunc{mw.Auth, mw.SessionOnly} {
		handler := SecurityHeaders(authenticated(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "private") }))
		req := httptest.NewRequest(http.MethodGet, "/api/account", nil)
		req.Header.Set("X-Auth-Token", login(t, admin, "admin", "password"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "same-origin" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("unsafe authenticated response: %d %v", w.Code, w.Header())
		}
	}
}

func TestAccessLogRedactsCredentialsWithoutChangingRequest(t *testing.T) {
	var output bytes.Buffer
	formatter := &redactedLogFormatter{base: &chimw.DefaultLogFormatter{Logger: log.New(&output, "", 0), NoColor: true}}
	for _, path := range []string{
		"/api/tunnels?api_key=secret-api-key",
		"/api/auth/oidc/callback?code=secret-auth-code&state=secret-state",
		"/status/secret-share-token",
		"/api/public/status/secret-share-token",
	} {
		handler := chimw.RequestLogger(formatter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.RequestURI != path || r.URL.RequestURI() != path {
				t.Fatal("log redaction changed the application request")
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	}
	if strings.Contains(output.String(), "secret-") || !strings.Contains(output.String(), "/api/tunnels") || !strings.Contains(output.String(), "204") {
		t.Fatalf("access log leaked credentials or lost diagnostics: %s", output.String())
	}
}
