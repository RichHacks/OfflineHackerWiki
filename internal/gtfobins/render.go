package gtfobins

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"strings"

	"hackerwiki/internal/mdrender"
)

func md(s string) template.HTML {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	out, err := mdrender.Render([]byte(s))
	if err != nil {
		return template.HTML(html.EscapeString(s))
	}
	return template.HTML(out)
}

func fieldset(legend string, body template.HTML) template.HTML {
	if body == "" {
		return ""
	}
	return template.HTML(fmt.Sprintf(`<fieldset><legend>%s</legend>%s</fieldset>`, html.EscapeString(legend), body))
}

func codeBlock(code string) template.HTML {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	return template.HTML(fmt.Sprintf(`<pre class="gb-code"><code>%s</code></pre>`, html.EscapeString(code)))
}

// IndexRow / IndexFunc feed both the server-rendered table and the client-side
// filter (embedded as JSON, mirroring the xss handler's pattern).
type IndexRow struct {
	Name      string      `json:"name"`
	Alias     string      `json:"alias,omitempty"`
	Functions []IndexFunc `json:"functions"`
}

type IndexFunc struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Contexts []string `json:"contexts"`
}

type LegendItem struct {
	Key         string
	Label       string
	Description string
}

type IndexPageData struct {
	Rows      []IndexRow
	RowsJSON  template.JS
	Total     int
	Functions []LegendItem
	Contexts  []LegendItem
}

func (d *Data) buildIndexFunctions(b *Binary) []IndexFunc {
	resolved := d.Resolve(b)
	var out []IndexFunc
	for _, key := range d.FuncOrder {
		examples, ok := resolved.Functions[key]
		if !ok {
			continue
		}
		ctxSet := map[string]bool{}
		for _, ex := range examples {
			if ex.Contexts == nil {
				if len(d.CtxOrder) > 0 {
					ctxSet[d.CtxOrder[0]] = true
				}
				continue
			}
			for ctx := range ex.Contexts {
				ctxSet[ctx] = true
			}
		}
		var ctxList []string
		for _, ctx := range d.CtxOrder {
			if ctxSet[ctx] {
				ctxList = append(ctxList, ctx)
			}
		}
		label := key
		if meta := d.FuncMeta[key]; meta != nil {
			label = meta.Label
		}
		out = append(out, IndexFunc{Key: key, Label: label, Contexts: ctxList})
	}
	return out
}

// BuildIndex builds the data for the filterable /gtfobins/ table.
func (d *Data) BuildIndex() IndexPageData {
	var rows []IndexRow
	for _, name := range d.Names {
		b := d.Binaries[name]
		row := IndexRow{Name: name, Alias: b.Alias, Functions: d.buildIndexFunctions(b)}
		rows = append(rows, row)
	}

	raw, _ := json.Marshal(rows)

	var funcs []LegendItem
	for _, key := range d.FuncOrder {
		if meta := d.FuncMeta[key]; meta != nil {
			funcs = append(funcs, LegendItem{Key: key, Label: meta.Label, Description: meta.Description})
		}
	}
	var ctxs []LegendItem
	for _, key := range d.CtxOrder {
		if meta := d.CtxMeta[key]; meta != nil {
			ctxs = append(ctxs, LegendItem{Key: key, Label: meta.Label, Description: meta.Description})
		}
	}

	return IndexPageData{
		Rows:      rows,
		RowsJSON:  template.JS(raw),
		Total:     len(rows),
		Functions: funcs,
		Contexts:  ctxs,
	}
}

// --- Detail page view models ---

type ContextView struct {
	ID       string
	Label    string
	Checked  bool
	BodyHTML template.HTML
}

type ExampleView struct {
	ID          string
	VersionHTML template.HTML
	CommentHTML template.HTML
	Contexts    []ContextView
	ExtraHTML   template.HTML
}

type FuncSection struct {
	Key         string
	Label       string
	Description template.HTML
	Examples    []ExampleView
}

type DetailPageData struct {
	Name        string
	IsAlias     bool
	AliasTarget string
	Comment     template.HTML
	Sections    []FuncSection
	SourceURL   string
	HistoryURL  string
	NotFound    bool
}

func (d *Data) buildContextView(exampleID string, ex Entry, ctxKey string, first bool) ContextView {
	ctxMeta := d.CtxMeta[ctxKey]
	var per *Context
	if ex.Contexts != nil {
		per = ex.Contexts[ctxKey]
	}

	var body strings.Builder
	if ctxMeta != nil {
		body.WriteString(string(md(ctxMeta.Description)))
	}

	switch ctxKey {
	case "sudo":
		if ctxMeta != nil {
			body.WriteString(string(fieldset("Remarks", md(ctxMeta.Extra.Environment))))
		}
	case "suid":
		if per != nil && per.Shell != nil && ctxMeta != nil {
			body.WriteString(string(fieldset("Remarks", md(ctxMeta.Extra.Shell[*per.Shell]))))
		}
	case "capabilities":
		if per != nil && len(per.List) > 0 && ctxMeta != nil {
			var caps []string
			for _, c := range per.List {
				caps = append(caps, "<code>"+html.EscapeString(c)+"</code>")
			}
			fmt.Fprintf(&body, "<p>%s %s.</p>", html.EscapeString(ctxMeta.Extra.List), strings.Join(caps, ", "))
		}
	}

	if per != nil && per.Comment != "" {
		body.WriteString(string(fieldset("Comment", md(per.Comment))))
	}

	code := ex.Code
	if per != nil && per.Code != "" {
		code = per.Code
	}
	body.WriteString(string(codeBlock(code)))

	label := ctxKey
	if ctxMeta != nil {
		label = ctxMeta.Label
	}

	return ContextView{
		ID:       exampleID + "-" + ctxKey,
		Label:    label,
		Checked:  first,
		BodyHTML: template.HTML(body.String()),
	}
}

func strOrCCLookup(known map[string]CodeComment, v *StrOrCC) (comment, code string) {
	if v == nil {
		return "", ""
	}
	if kc, ok := known[v.Key]; ok {
		comment, code = kc.Comment, kc.Code
	}
	if comment == "" {
		comment = v.Comment
	}
	if code == "" {
		code = v.Code
	}
	return comment, code
}

func (d *Data) buildExtraHTML(funcKey string, ex Entry) template.HTML {
	meta := d.FuncMeta[funcKey]
	var buf strings.Builder

	switch funcKey {
	case "shell", "command", "reverse-shell", "bind-shell":
		if ex.Blind != nil && *ex.Blind && meta != nil {
			buf.WriteString(string(fieldset("Output", md(meta.Extra.Blind[true]))))
		}
	}
	switch funcKey {
	case "file-write", "file-read", "upload", "download":
		if ex.Binary != nil && !*ex.Binary && meta != nil {
			buf.WriteString(string(fieldset("Remarks", md(meta.Extra.Binary[false]))))
		}
	}
	switch funcKey {
	case "shell", "reverse-shell", "bind-shell":
		if ex.TTY != nil && !*ex.TTY && meta != nil {
			buf.WriteString(string(fieldset("TTY", md(meta.Extra.TTY[false]))))
		}
	}

	if meta != nil {
		switch funcKey {
		case "reverse-shell":
			if ex.Listener != nil {
				comment, code := strOrCCLookup(meta.Extra.Listener, ex.Listener)
				buf.WriteString(string(fieldset("Listener", md(comment)+codeBlock(code))))
			}
		case "bind-shell":
			if ex.Connector != nil {
				comment, code := strOrCCLookup(meta.Extra.Connector, ex.Connector)
				buf.WriteString(string(fieldset("Connector", md(comment)+codeBlock(code))))
			}
		case "upload":
			if ex.Receiver != nil {
				comment, code := strOrCCLookup(meta.Extra.Receiver, ex.Receiver)
				buf.WriteString(string(fieldset("Receiver", md(comment)+codeBlock(code))))
			}
		case "download":
			if ex.Sender != nil {
				comment, code := strOrCCLookup(meta.Extra.Sender, ex.Sender)
				buf.WriteString(string(fieldset("Sender", md(comment)+codeBlock(code))))
			}
		case "library-load":
			buf.WriteString(string(fieldset("Payload", md(meta.Extra.Payload))))
		}
	}

	if funcKey == "inherit" && ex.From != "" {
		if target, ok := d.Binaries[ex.From]; ok {
			resolved := d.Resolve(target)
			var pills []string
			for _, fk := range d.FuncOrder {
				if fm, ok := resolved.Functions[fk]; ok && len(fm) > 0 {
					label := fk
					if lm := d.FuncMeta[fk]; lm != nil {
						label = lm.Label
					}
					pills = append(pills, fmt.Sprintf(`<a class="gb-pill" href="/gtfobins/%s#%s">%s</a>`,
						html.EscapeString(ex.From), html.EscapeString(fk), html.EscapeString(label)))
				}
			}
			fmt.Fprintf(&buf,
				`<fieldset><legend>Functions</legend><p>Inherits from <a href="/gtfobins/%s"><code>%s</code></a>, thus possibly granting the following functions:</p><p>%s</p></fieldset>`,
				html.EscapeString(ex.From), html.EscapeString(ex.From), strings.Join(pills, " "))
		}
	}

	return template.HTML(buf.String())
}

func (d *Data) buildExampleView(funcKey string, idx int, ex Entry) ExampleView {
	id := fmt.Sprintf("%s-%d", funcKey, idx+1)

	var ctxViews []ContextView
	first := true
	for _, ctxKey := range d.CtxOrder {
		present := false
		if ex.Contexts != nil {
			_, present = ex.Contexts[ctxKey]
		} else {
			present = len(d.CtxOrder) > 0 && ctxKey == d.CtxOrder[0]
		}
		if !present {
			continue
		}
		ctxViews = append(ctxViews, d.buildContextView(id, ex, ctxKey, first))
		first = false
	}

	return ExampleView{
		ID:          id,
		VersionHTML: fieldset("Version requirements", md(ex.Version)),
		CommentHTML: fieldset("Comment", md(ex.Comment)),
		Contexts:    ctxViews,
		ExtraHTML:   d.buildExtraHTML(funcKey, ex),
	}
}

// BuildDetail builds the view model for a single binary's page. ok is false if the
// name doesn't exist in the dataset.
func (d *Data) BuildDetail(name string) (DetailPageData, bool) {
	b, ok := d.Binaries[name]
	if !ok {
		return DetailPageData{}, false
	}
	resolved := d.Resolve(b)

	page := DetailPageData{
		Name:       name,
		SourceURL:  "https://github.com/GTFOBins/GTFOBins.github.io/blob/master/_gtfobins/" + name,
		HistoryURL: "https://github.com/GTFOBins/GTFOBins.github.io/commits/master/_gtfobins/" + name,
	}
	if b.Alias != "" {
		page.IsAlias = true
		page.AliasTarget = b.Alias
	}
	page.Comment = fieldset("Comment", md(resolved.Comment))

	for _, key := range d.FuncOrder {
		examples, ok := resolved.Functions[key]
		if !ok {
			continue
		}
		meta := d.FuncMeta[key]
		section := FuncSection{Key: key}
		if meta != nil {
			section.Label = meta.Label
			section.Description = md(meta.Description)
		} else {
			section.Label = key
		}
		for i, ex := range examples {
			section.Examples = append(section.Examples, d.buildExampleView(key, i, ex))
		}
		page.Sections = append(page.Sections, section)
	}

	return page, true
}
