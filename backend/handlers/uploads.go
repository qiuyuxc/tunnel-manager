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

	"tunnel-manager/store"
)

// allowed image extensions for uploads (status-page icons and avatars).
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
	url, err := h.saveImage(w, r)
	if err != nil {
		if isUploadOSError(err) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// UploadAvatar handles POST /api/account/avatar with a multipart "file"
// field: it stores the image like other uploads and saves the returned URL
// on the authenticated account.
func (h *UploadsHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "avatar storage unavailable"})
		return
	}
	url, err := h.saveImage(w, r)
	if err != nil {
		if isUploadOSError(err) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return
	}
	stored, ok := h.store.GetUserByID(user.ID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if err := h.store.SetUserProfile(user.ID, stored.Nickname, url); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unable to save avatar"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// saveImage reads one multipart "file" field, validates it as an allowed
// image and persists it under the uploads directory.
func (h *UploadsHandler) saveImage(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		return "", errTooLarge
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		return "", errMissingFile
	}
	defer f.Close()

	name := r.FormValue("filename")
	ext := strings.ToLower(filepath.Ext(name))
	if _, ok := uploadExt[ext]; !ok {
		return "", errBadExt
	}

	body, err := io.ReadAll(f)
	if err != nil || len(body) == 0 {
		return "", errEmptyFile
	}
	sum := sha256.Sum256(body)
	outName := hex.EncodeToString(sum[:8]) + ext
	dir := h.Dir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", &uploadOSError{err}
	}
	if err := os.WriteFile(filepath.Join(dir, outName), body, 0o644); err != nil {
		return "", &uploadOSError{err}
	}
	return "/uploads/" + outName, nil
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

// UploadsHandler stores and serves small images.
type UploadsHandler struct {
	Dir   string
	store *store.Store
}

// NewUploadsHandler builds a handler rooted at dir.
func NewUploadsHandler(dir string) *UploadsHandler { return &UploadsHandler{Dir: dir} }

// SetStore wires the uploads handler to the store so avatar uploads can
// persist the resulting URL on the account.
func (h *UploadsHandler) SetStore(st *store.Store) { h.store = st }

var (
	errTooLarge    = &uploadError{"file too large (max 4 MiB)"}
	errMissingFile = &uploadError{"missing file field"}
	errBadExt      = &uploadError{"only png/jpg/jpeg/gif/webp allowed"}
	errEmptyFile   = &uploadError{"empty file"}
)

type uploadError struct{ msg string }

func (e *uploadError) Error() string { return e.msg }

// uploadOSError wraps a filesystem failure that should surface as a 500.
type uploadOSError struct{ err error }

func (e *uploadOSError) Error() string { return e.err.Error() }

// isUploadOSError reports whether err is a filesystem-level failure.
func isUploadOSError(err error) bool {
	_, ok := err.(*uploadOSError)
	return ok
}
