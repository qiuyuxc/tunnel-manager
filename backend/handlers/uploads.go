package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// allowed image extensions for status-page icon uploads.
var uploadExt = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

const maxUploadBytes = 4 << 20 // 4 MiB

// UploadImage handles POST /api/uploads with a multipart "file" field and
// returns {url} usable as the monitor public_icon.
func (h *UploadsHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large (max 4 MiB)"})
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer f.Close()

	name := r.FormValue("filename")
	ext := strings.ToLower(filepath.Ext(name))
	if _, ok := uploadExt[ext]; !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only png/jpg/jpeg/gif/webp allowed"})
		return
	}

	body, err := io.ReadAll(f)
	if err != nil || len(body) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty file"})
		return
	}
	sum := sha256.Sum256(body)
	outName := hex.EncodeToString(sum[:8]) + ext
	dir := h.Dir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := os.WriteFile(filepath.Join(dir, outName), body, 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"url": "/uploads/" + outName})
}

// Serve serves stored uploads; mounted before the SPA catch-all.
func (h *UploadsHandler) Serve(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(chi.URLParam(r, "*"))
	if name == "." || name == "/" {
		http.NotFound(w, r)
		return
	}
	p := filepath.Join(h.Dir, name)
	if _, err := os.Stat(p); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, p)
}

// UploadsHandler stores and serves small status-page images.
type UploadsHandler struct {
	Dir string
}

// NewUploadsHandler builds a handler rooted at dir.
func NewUploadsHandler(dir string) *UploadsHandler { return &UploadsHandler{Dir: dir} }
