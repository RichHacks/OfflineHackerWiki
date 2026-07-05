package wiki

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"hackerwiki/internal/mdrender"
)

type searchEntry struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type indexEntry struct {
	title string
	url   string
	text  string
}

func (h *Handler) warmIndex() {
	h.idxOnce.Do(func() {
		var entries []indexEntry
		_ = filepath.Walk(h.srcPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(h.srcPath, path)
			rel = strings.TrimSuffix(rel, filepath.Ext(rel))
			url := h.prefix + "/" + filepath.ToSlash(rel)

			title := mdrender.PageTitle(raw)
			if title == "" {
				title = filepath.Base(rel)
			}
			entries = append(entries, indexEntry{
				title: title,
				url:   url,
				text:  wikiPlaintext(raw),
			})
			return nil
		})
		h.idxCache = entries
	})
}

func (h *Handler) searchHandler(w http.ResponseWriter, r *http.Request) {
	h.warmIndex()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var results []searchEntry

	if q != "" {
		lower := strings.ToLower(q)
		type scored struct {
			e     searchEntry
			score int
		}
		var hits []scored

		for _, e := range h.idxCache {
			tl := strings.ToLower(e.title)
			tx := strings.ToLower(e.text)
			sc := 0
			if tl == lower {
				sc = 100
			} else if strings.HasPrefix(tl, lower) {
				sc = 80
			} else if strings.Contains(tl, lower) {
				sc = 60
			}
			if strings.Contains(tx, lower) {
				if sc == 0 {
					sc = 40
				}
				sc += strings.Count(tx, lower)
			}
			if sc == 0 {
				continue
			}
			hits = append(hits, scored{
				e: searchEntry{
					Title:   e.title,
					URL:     e.url,
					Snippet: wikiSnippet(e.text, q, 200),
				},
				score: sc,
			})
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
		for _, h := range hits {
			results = append(results, h.e)
		}
	}

	if len(results) > 30 {
		results = results[:30]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func wikiSnippet(text, query string, maxLen int) string {
	if text == "" {
		return ""
	}
	text = strings.Join(strings.Fields(text), " ")
	if query == "" {
		return truncate(text, maxLen)
	}
	idx := strings.Index(strings.ToLower(text), strings.ToLower(query))
	if idx < 0 {
		return truncate(text, maxLen)
	}
	start := idx - 80
	if start < 0 {
		start = 0
	}
	for start > 0 && text[start] != ' ' {
		start--
	}
	end := start + maxLen
	if end > len(text) {
		end = len(text)
	}
	result := text[start:end]
	if !utf8.ValidString(result) {
		result = strings.ToValidUTF8(result, "")
	}
	if start > 0 {
		result = "…" + strings.TrimSpace(result)
	}
	if end < len(text) {
		result = strings.TrimSpace(result) + "…"
	}
	return result
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func wikiPlaintext(src []byte) string {
	// Strip mdBook directives
	lines := strings.Split(string(src), "\n")
	var out []string
	for _, l := range lines {
		if !strings.Contains(l, "{{#") {
			out = append(out, l)
		}
	}
	cleaned := []byte(strings.Join(out, "\n"))
	html, err := mdrender.Render(cleaned)
	if err != nil {
		return string(src)
	}
	return stripTags(html)
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
			b.WriteRune(' ')
		case !in:
			b.WriteRune(r)
		}
	}
	return b.String()
}
