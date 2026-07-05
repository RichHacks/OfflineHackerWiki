package xss

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Handler serves the XSS cheat sheet.
type Handler struct {
	dataPath string
	tmpl     *template.Template
	once     sync.Once
	data     *PageData
	dataErr  error
}

// XSSRow is a single filterable row in the cheat sheet table.
type XSSRow struct {
	Event       string   `json:"event"`
	Description string   `json:"description"`
	Tag         string   `json:"tag"`
	Code        string   `json:"code"`
	Browsers    []string `json:"browsers"`
	Interaction bool     `json:"interaction"`
	Section     string   `json:"section"` // "events-auto", "events-user", or a category name
}

// PageData is everything passed to the template.
type PageData struct {
	RowsJSON  template.JS // JSON-encoded []XSSRow — template.JS so html/template doesn't double-escape
	Tags      []string
	Events    []string
	TotalRows int
}

func New(dataPath string, tmpl *template.Template) *Handler {
	return &Handler{dataPath: dataPath, tmpl: tmpl}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(func() {
		h.data, h.dataErr = h.buildData()
	})
	if h.dataErr != nil {
		http.Error(w, "failed to load XSS data: "+h.dataErr.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "xss.html", h.data); err != nil {
		log.Printf("xss template error: %v", err)
	}
}

func (h *Handler) buildData() (*PageData, error) {
	var rows []XSSRow
	tagSet := map[string]bool{}
	eventSet := map[string]bool{}

	if raw, err := os.ReadFile(filepath.Join(h.dataPath, "events.json")); err == nil {
		var events map[string]struct {
			Description string `json:"description"`
			Tags        []struct {
				Tag         string   `json:"tag"`
				Code        string   `json:"code"`
				Browsers    []string `json:"browsers"`
				Interaction bool     `json:"interaction"`
			} `json:"tags"`
		}
		if err := json.Unmarshal(raw, &events); err == nil {
			names := sortedKeys(events)
			for _, name := range names {
				e := events[name]
				eventSet[name] = true
				for _, t := range e.Tags {
					section := "events-auto"
					if t.Interaction {
						section = "events-user"
					}
					tag := normalizeTag(t.Tag)
					if tag != "*" {
						tagSet[tag] = true
					}
					rows = append(rows, XSSRow{
						Event:       name,
						Description: e.Description,
						Tag:         tag,
						Code:        t.Code,
						Browsers:    normBrowsers(t.Browsers),
						Interaction: t.Interaction,
						Section:     section,
					})
				}
			}
		}
	}

	stdCats := []struct{ name, file string }{
		{"Classic vectors", "classic.json"},
		{"Useful tags", "useful_tags.json"},
		{"Frameworks", "frameworks.json"},
		{"Obfuscation", "obfuscation.json"},
		{"Polyglots", "polyglot.json"},
		{"Protocols", "protocols.json"},
		{"Dangling markup", "dangling_markup.json"},
		{"Encodings", "encodings.json"},
		{"WAF bypass", "waf_bypass_global_obj.json"},
		{"Special tags", "special_tags.json"},
		{"Content types", "content-types.json"},
		{"Consuming tags", "consuming_tags.json"},
		{"File uploads", "file_uploads.json"},
		{"Restricted characters", "restricted_characters.json"},
		{"Response content types", "response-content-types.json"},
	}
	for _, cat := range stdCats {
		raw, err := os.ReadFile(filepath.Join(h.dataPath, cat.file))
		if err != nil {
			continue
		}
		var entries []struct {
			Description string   `json:"description"`
			Code        string   `json:"code"`
			Browsers    []string `json:"browsers"`
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			continue
		}
		for _, e := range entries {
			rows = append(rows, XSSRow{
				Description: e.Description,
				Code:        e.Code,
				Browsers:    normBrowsers(e.Browsers),
				Section:     "cat:" + cat.name,
			})
		}
	}

	if raw, err := os.ReadFile(filepath.Join(h.dataPath, "angularjs.json")); err == nil {
		var entries []struct {
			VersionRange string          `json:"versionRange"`
			Version      string          `json:"version"`
			Authors      json.RawMessage `json:"authors"` // []object, skip
			Vector       string          `json:"vector"`
		}
		if json.Unmarshal(raw, &entries) == nil {
			for _, e := range entries {
				desc := e.VersionRange
				if desc == "" {
					desc = e.Version
				}
				rows = append(rows, XSSRow{
					Description: desc,
					Code:        e.Vector,
					Section:     "cat:AngularJS",
				})
			}
		}
	}

	if raw, err := os.ReadFile(filepath.Join(h.dataPath, "vuejs.json")); err == nil {
		var entries []struct {
			VersionRange *string         `json:"versionRange"` // nullable
			Version      json.Number     `json:"version"`      // integer in this file
			Authors      json.RawMessage `json:"authors"`
			Vector       string          `json:"vector"`
		}
		if json.Unmarshal(raw, &entries) == nil {
			for _, e := range entries {
				desc := "v" + e.Version.String()
				if e.VersionRange != nil && *e.VersionRange != "" {
					desc = *e.VersionRange
				}
				rows = append(rows, XSSRow{
					Description: desc,
					Code:        e.Vector,
					Section:     "cat:Vue.js",
				})
			}
		}
	}

	if raw, err := os.ReadFile(filepath.Join(h.dataPath, "prototype-pollution.json")); err == nil {
		var entries []struct {
			Library     string `json:"library"`
			Version     string `json:"version"`
			Payload     string `json:"payload"`
			Fingerprint string `json:"fingerprint"`
		}
		if json.Unmarshal(raw, &entries) == nil {
			for _, e := range entries {
				desc := e.Library
				if e.Version != "" {
					desc += " " + e.Version
				}
				rows = append(rows, XSSRow{
					Description: desc,
					Code:        e.Payload,
					Section:     "cat:Prototype pollution",
				})
			}
		}
	}

	tags := sortedSetKeys(tagSet)
	events := sortedSetKeys(eventSet)

	rowsJSON, _ := json.Marshal(rows)
	return &PageData{
		RowsJSON:  template.JS(rowsJSON),
		Tags:      tags,
		Events:    events,
		TotalRows: len(rows),
	}, nil
}

var (
	headingTagRe    = regexp.MustCompile(`^h[1-6]$`)
	trailingDigitRe = regexp.MustCompile(`[0-9]+$`)
)

// normalizeTag strips the numeric suffix the source data uses to disambiguate
// multiple payloads for the same tag (e.g. "a2", "img2", "input3"). Real
// heading tags (h1-h6) are left untouched since the digit is meaningful there.
func normalizeTag(tag string) string {
	if tag == "" || tag == "*" || headingTagRe.MatchString(tag) {
		return tag
	}
	return trailingDigitRe.ReplaceAllString(tag, "")
}

func normBrowsers(in []string) []string {
	out := make([]string, len(in))
	for i, b := range in {
		out[i] = strings.ToLower(b)
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedSetKeys(m map[string]bool) []string {
	return sortedKeys(m)
}

// TemplateFuncs returns functions needed by the XSS template.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"lower": strings.ToLower,
	}
}
