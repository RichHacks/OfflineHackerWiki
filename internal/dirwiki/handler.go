package dirwiki

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"hackerwiki/internal/mdrender"
)

// Handler serves a directory-per-topic wiki (PayloadsAllTheThings style).
// Each top-level directory is a topic; its README.md is the main content.
type Handler struct {
	name     string
	rootPath string
	prefix   string
	tmpl     *template.Template
	tmplName string
	idxOnce  sync.Once
	idxCache []indexEntry
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
		tmplName: "payloads.html",
	}
}

// Mux returns an http.Handler that wires the search endpoint alongside page serving.
func (h *Handler) Mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_search", h.searchHandler)
	mux.Handle("/", h)
	return mux
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if urlPath == "" || urlPath == "/" {
		// Serve the root README.md if it exists
		urlPath = "/README"
	}

	cleaned := filepath.Clean(urlPath)
	if strings.Contains(cleaned, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	absPath := filepath.Join(h.rootPath, cleaned)

	// Determine what to serve
	var filePath string
	var isDir bool

	if info, err := os.Stat(absPath); err == nil {
		if info.IsDir() {
			isDir = true
			filePath = filepath.Join(absPath, "README.md")
		} else {
			filePath = absPath
		}
	} else if _, err := os.Stat(absPath + ".md"); err == nil {
		filePath = absPath + ".md"
	}

	if filePath == "" {
		http.NotFound(w, r)
		return
	}

	// Serve non-markdown files directly
	if !strings.EqualFold(filepath.Ext(filePath), ".md") {
		http.ServeFile(w, r, filePath)
		return
	}

	src, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	content, err := mdrender.Render(src)
	if err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	currentURL := h.prefix + cleaned
	topics := h.listTopics()
	nav := h.buildNav(topics, currentURL, isDir, cleaned)

	title := mdrender.PageTitle(src)
	if title == "" {
		title = strings.Trim(cleaned, "/")
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
	if err := h.tmpl.ExecuteTemplate(w, h.tmplName, data); err != nil {
		log.Printf("dirwiki template error: %v", err)
	}
}

// listTopics returns all top-level topic directories.
func (h *Handler) listTopics() []string {
	entries, err := os.ReadDir(h.rootPath)
	if err != nil {
		return nil
	}
	var topics []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(e.Name(), "_") {
			topics = append(topics, e.Name())
		}
	}
	sort.Strings(topics)
	return topics
}

func (h *Handler) buildNav(topics []string, currentURL string, isDir bool, cleaned string) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<ul class="nav-tree">`)

	// Root link
	rootCls := ""
	if currentURL == h.prefix || currentURL == h.prefix+"/README" {
		rootCls = " active"
	}
	sb.WriteString(`<li class="nav-item"><a class="nav-link` + rootCls + `" href="` + h.prefix + `/">Overview</a></li>`)
	sb.WriteString(`<li class="nav-section"><span class="nav-section-title">Topics</span><ul class="nav-children open">`)

	for _, topic := range topics {
		// Compare using decoded path (currentURL is already URL-decoded by net/http)
		topicPathDecoded := h.prefix + "/" + topic
		topicURL := h.prefix + "/" + urlEncode(topic)
		active := currentURL == topicPathDecoded || strings.HasPrefix(currentURL, topicPathDecoded+"/")
		cls := "nav-item"
		linkCls := "nav-link"
		if active {
			cls += " open"
			linkCls += " active"
		}
		sb.WriteString(`<li class="` + cls + `"><a class="` + linkCls + `" href="` + topicURL + `">` + template.HTMLEscapeString(topic) + `</a>`)

		// Show sub-files if this topic is active
		if active {
			subFiles := h.listSubFiles(topic)
			if len(subFiles) > 0 {
				sb.WriteString(`<ul class="nav-children">`)
				for _, sf := range subFiles {
					sfDecoded := topicPathDecoded + "/" + strings.TrimSuffix(sf, ".md")
					sfURL := topicURL + "/" + urlEncode(strings.TrimSuffix(sf, ".md"))
					sfCls := "nav-link"
					if sfDecoded == currentURL {
						sfCls += " active"
					}
					label := strings.TrimSuffix(sf, ".md")
					sb.WriteString(`<li class="nav-item"><a class="` + sfCls + `" href="` + sfURL + `">` + template.HTMLEscapeString(label) + `</a></li>`)
				}
				sb.WriteString(`</ul>`)
			}
		}
		sb.WriteString(`</li>`)
	}

	sb.WriteString(`</ul></li>`)
	sb.WriteString(`</ul>`)
	return template.HTML(sb.String())
}

func (h *Handler) listSubFiles(topic string) []string {
	dir := filepath.Join(h.rootPath, topic)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && !strings.EqualFold(e.Name(), "README.md") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files
}

// urlEncode encodes a path component (spaces → %20, etc.) without encoding slashes.
func urlEncode(s string) string {
	s = strings.ReplaceAll(s, " ", "%20")
	s = strings.ReplaceAll(s, "#", "%23")
	return s
}
