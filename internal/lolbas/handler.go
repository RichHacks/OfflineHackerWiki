package lolbas

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Handler serves LOLBAS: a filterable index at "/" and one detail page per
// binary at "/<Category>/<Slug>" (mount this under a prefix such as "/lolbas/"
// and http.StripPrefix it, same as the other content handlers).
type Handler struct {
	dataPath string
	tmpl     *template.Template
	once     sync.Once
	data     *Data
	err      error
}

func New(dataPath string, tmpl *template.Template) *Handler {
	return &Handler{dataPath: dataPath, tmpl: tmpl}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(func() {
		h.data, h.err = Load(h.dataPath)
	})
	if h.err != nil {
		http.Error(w, "failed to load LOLBAS data: "+h.err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		index := h.data.BuildIndex()
		if err := h.tmpl.ExecuteTemplate(w, "lolbas.html", index); err != nil {
			log.Printf("lolbas index template error: %v", err)
		}
		return
	}

	category, slug, ok := strings.Cut(path, "/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	page, ok := h.data.BuildDetail(category, slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.tmpl.ExecuteTemplate(w, "lolbas-bin.html", page); err != nil {
		log.Printf("lolbas detail template error: %v", err)
	}
}
