package wiki

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"hackerwiki/internal/mdrender"
)

// Handler serves an mdBook-style wiki (SUMMARY.md + markdown files in src/).
type Handler struct {
	name     string // display name
	rootPath string // path to wiki repo root (SUMMARY.md is at src/SUMMARY.md)
	srcPath  string // rootPath/src
	prefix   string // URL prefix, e.g. "/hacktricks"
	tmpl     *template.Template

	once    sync.Once
	navTree []*NavNode

	idxOnce  sync.Once
	idxCache []indexEntry
}

// WikiPage is the template data for a rendered wiki page.
type WikiPage struct {
	Title      string
	WikiName   string
	Prefix     string
	Nav        template.HTML
	Content    template.HTML
	CurrentURL string
}

func New(prefix, name, rootPath string, tmpl *template.Template) *Handler {
	h := &Handler{
		name:     name,
		rootPath: rootPath,
		srcPath:  filepath.Join(rootPath, "src"),
		prefix:   prefix,
		tmpl:     tmpl,
	}
	go h.warmIndex() // pre-build search index in background
	return h
}

// Mux returns an http.Handler with the search endpoint wired alongside page serving.
func (h *Handler) Mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_search", h.searchHandler)
	mux.Handle("/", h)
	return mux
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(func() {
		summaryPath := filepath.Join(h.srcPath, "SUMMARY.md")
		h.navTree = parseSummary(summaryPath, h.prefix)
	})

	// r.URL.Path has already had the prefix stripped by http.StripPrefix
	urlPath := r.URL.Path
	if urlPath == "" || urlPath == "/" {
		urlPath = "/README"
	}

	// Try to resolve to a file on disk
	filePath, isMarkdown := h.resolve(urlPath)
	if filePath == "" {
		http.NotFound(w, r)
		return
	}

	if !isMarkdown {
		http.ServeFile(w, r, filePath)
		return
	}

	src, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	content, err := renderMarkdown(src)
	if err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	currentURL := h.prefix + urlPath
	nav := RenderNav(h.navTree, currentURL)

	title := mdrender.PageTitle(src)
	if title == "" {
		title = strings.Trim(urlPath, "/")
	}

	data := WikiPage{
		Title:      title + " — " + h.name,
		WikiName:   h.name,
		Prefix:     h.prefix,
		Nav:        nav,
		Content:    template.HTML(content),
		CurrentURL: currentURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "wiki.html", data); err != nil {
		log.Printf("wiki template error: %v", err)
	}
}

// resolve maps a URL path to an absolute file path.
// Returns ("", false) if not found.
func (h *Handler) resolve(urlPath string) (string, bool) {
	// Clean path and prevent traversal
	cleaned := filepath.Clean(urlPath)
	if strings.Contains(cleaned, "..") {
		return "", false
	}

	base := filepath.Join(h.srcPath, cleaned)

	// 1. Exact match (for non-markdown static assets)
	if info, err := os.Stat(base); err == nil && !info.IsDir() {
		return base, strings.EqualFold(filepath.Ext(base), ".md")
	}

	// 2. Try adding .md
	withMD := base + ".md"
	if _, err := os.Stat(withMD); err == nil {
		return withMD, true
	}

	// 3. Try as directory/README.md
	readmePath := filepath.Join(base, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		return readmePath, true
	}

	return "", false
}

