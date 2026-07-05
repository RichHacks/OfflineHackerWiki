package dirwiki

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
	Topic   string `json:"topic"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type indexEntry struct {
	topic string
	url   string
	text  string // full plaintext content
}

func (h *Handler) buildIndex() []indexEntry {
	h.idxOnce.Do(func() {
		topics := h.listTopics()
		var entries []indexEntry

		for _, topic := range topics {
			topicURL := h.prefix + "/" + urlEncode(topic)

			// Index README.md
			readmePath := filepath.Join(h.rootPath, topic, "README.md")
			if raw, err := os.ReadFile(readmePath); err == nil {
				entries = append(entries, indexEntry{
					topic: topic,
					url:   topicURL,
					text:  extractText(raw),
				})
			} else {
				entries = append(entries, indexEntry{topic: topic, url: topicURL, text: ""})
			}

			// Index every .md sub-file with its full content
			for _, sf := range h.listSubFiles(topic) {
				sfPath := filepath.Join(h.rootPath, topic, sf)
				sfTitle := strings.TrimSuffix(sf, ".md")
				sfURL := topicURL + "/" + urlEncode(sfTitle)
				sfLabel := topic + " › " + sfTitle

				if raw, err := os.ReadFile(sfPath); err == nil {
					entries = append(entries, indexEntry{
						topic: sfLabel,
						url:   sfURL,
						text:  extractText(raw),
					})
				}
			}
		}
		h.idxCache = entries
	})
	return h.idxCache
}

// searchHandler serves GET /_search?q=<query> as a JSON array of searchEntry.
func (h *Handler) searchHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var results []searchEntry

	if q == "" {
		// Return all topics (no sub-files) so the client can pre-populate on focus
		for _, e := range h.buildIndex() {
			if !strings.Contains(e.topic, " › ") {
				results = append(results, searchEntry{
					Topic:   e.topic,
					URL:     e.url,
					Snippet: snippet(e.text, "", 180),
				})
			}
		}
	} else {
		lower := strings.ToLower(q)
		type scoredEntry struct {
			entry searchEntry
			score int
		}
		var scored []scoredEntry

		for _, e := range h.buildIndex() {
			topicLow := strings.ToLower(e.topic)
			textLow := strings.ToLower(e.text)

			score := 0
			// Topic name match = highest priority
			if topicLow == lower {
				score = 100
			} else if strings.HasPrefix(topicLow, lower) {
				score = 80
			} else if strings.Contains(topicLow, lower) {
				score = 60
			}
			// Full-text match
			if strings.Contains(textLow, lower) {
				if score == 0 {
					score = 40
				}
				score += strings.Count(textLow, lower) // more occurrences = more relevant
			}

			if score == 0 {
				continue
			}

			scored = append(scored, scoredEntry{
				entry: searchEntry{
					Topic:   e.topic,
					URL:     e.url,
					Snippet: snippet(e.text, q, 220),
				},
				score: score,
			})
		}

		sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
		for _, s := range scored {
			results = append(results, s.entry)
		}
	}

	// Cap at 30 results
	if len(results) > 30 {
		results = results[:30]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// snippet finds the first occurrence of query in text and returns surrounding context.
// If query is empty, returns the first maxLen chars.
func snippet(text, query string, maxLen int) string {
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

	// Centre window around match
	start := idx - 80
	if start < 0 {
		start = 0
	}
	// Snap to word boundary
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

// extractText renders markdown to plain text, stripping HTML tags.
func extractText(src []byte) string {
	cleaned := stripDirectives(src)
	html, err := mdrender.Render(cleaned)
	if err != nil {
		return string(src)
	}
	return stripHTML(html)
}

func stripDirectives(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	var out []string
	for _, line := range lines {
		if !strings.Contains(line, "{{#") {
			out = append(out, line)
		}
	}
	return []byte(strings.Join(out, "\n"))
}

func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			b.WriteRune(' ')
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
