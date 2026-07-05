package staticsite

import (
	"net/http"
	"os"
)

// Handler serves a pre-built static website from a directory on disk.
// Requests for "/" serve index.html; all other paths are served as static files.
type Handler struct {
	rootPath string
	fs       http.Handler
}

func New(rootPath string) *Handler {
	return &Handler{
		rootPath: rootPath,
		fs:       http.FileServer(http.Dir(rootPath)),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(h.rootPath); err != nil {
		http.Error(w, "content not found", http.StatusNotFound)
		return
	}
	h.fs.ServeHTTP(w, r)
}
