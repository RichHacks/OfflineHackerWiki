package flatwiki

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Handler serves a flat collection of markdown files (GitHub wiki style).
// No SUMMARY.md — nav is generated from the file listing.
type Handler struct {
	name     string
	rootPath string
	prefix   string
	tmpl     *template.Template
}

type WikiPage struct {
	Title      string
	WikiName   string
	Prefix     string
	Nav        template.HTML
	Content    template.HTML
	CurrentURL string
}

func New(prefix, name, rootPath string, tmpl *template.Template) *Handler {
	return &Handler{
		name:     name,
		rootPath: rootPath,
		prefix:   prefix,
		tmpl:     tmpl,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if urlPath == "" || urlPath == "/" {
		urlPath = "/Home"
	}

	cleaned := filepath.Clean(urlPath)
	if strings.Contains(cleaned, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	filePath := h.resolve(cleaned)
	if filePath == "" {
		http.NotFound(w, r)
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

	currentURL := h.prefix + cleaned
	nav := h.buildNav(currentURL)

	data := WikiPage{
		Title:      strings.TrimSuffix(filepath.Base(cleaned), filepath.Ext(cleaned)) + " — " + h.name,
		WikiName:   h.name,
		Prefix:     h.prefix,
		Nav:        nav,
		Content:    template.HTML(content),
		CurrentURL: currentURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "wiki.html", data); err != nil {
		log.Printf("flatwiki template error: %v", err)
	}
}

func (h *Handler) resolve(urlPath string) string {
	base := filepath.Join(h.rootPath, urlPath)

	// Exact .md file
	if _, err := os.Stat(base + ".md"); err == nil {
		return base + ".md"
	}
	// Already has extension
	if _, err := os.Stat(base); err == nil {
		return base
	}
	return ""
}

func (h *Handler) buildNav(currentURL string) template.HTML {
	entries, err := os.ReadDir(h.rootPath)
	if err != nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<ul class="nav-tree">`)

	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, "_") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	// Home first
	for _, name := range files {
		if strings.EqualFold(name, "home.md") {
			linkPath := h.prefix + "/" + strings.TrimSuffix(name, ".md")
			cls := "nav-link"
			if linkPath == currentURL {
				cls += " active"
			}
			sb.WriteString(`<li class="nav-item"><a class="` + cls + `" href="` + linkPath + `">Home</a></li>`)
			break
		}
	}

	for _, name := range files {
		if strings.EqualFold(name, "home.md") {
			continue
		}
		title := navTitle(strings.TrimSuffix(name, ".md"))
		linkPath := h.prefix + "/" + strings.TrimSuffix(name, ".md")
		cls := "nav-link"
		if linkPath == currentURL {
			cls += " active"
		}
		sb.WriteString(`<li class="nav-item"><a class="` + cls + `" href="` + template.HTMLEscapeString(linkPath) + `">` + template.HTMLEscapeString(title) + `</a></li>`)
	}

	sb.WriteString(`</ul>`)
	return template.HTML(sb.String())
}

// navTitle converts a filename stem (without .md) to a human-readable nav label.
// Handles Certipy's "01-‐-Introduction" style names.
func navTitle(stem string) string {
	stem = strings.ReplaceAll(stem, "‐", "-")
	if len(stem) >= 3 && stem[0] >= '0' && stem[0] <= '9' && stem[1] >= '0' && stem[1] <= '9' && stem[2] == '-' {
		stem = strings.TrimLeft(stem[3:], "- ")
	}
	return strings.TrimSpace(stem)
}
