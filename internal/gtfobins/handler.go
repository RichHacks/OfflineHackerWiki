package gtfobins

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Handler serves GTFOBins: a filterable index at "/" and one detail page per
// binary at "/<name>" (mount this under a prefix such as "/gtfobins/" and
// http.StripPrefix it, same as the other content handlers).
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
		http.Error(w, "failed to load GTFOBins data: "+h.err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		index := h.data.BuildIndex()
		if err := h.tmpl.ExecuteTemplate(w, "gtfobins.html", index); err != nil {
			log.Printf("gtfobins index template error: %v", err)
		}
		return
	}

	page, ok := h.data.BuildDetail(path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.tmpl.ExecuteTemplate(w, "gtfobins-bin.html", page); err != nil {
		log.Printf("gtfobins detail template error: %v", err)
	}
}
