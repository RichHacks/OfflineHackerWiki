package lolbas

import (
	"encoding/json"
	"html"
	"html/template"
	"sort"
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

// --- Index page ---

type IndexRow struct {
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Category  string   `json:"category"`
	Functions []string `json:"functions"`
	Mitre     []string `json:"mitre"`
}

type IndexPageData struct {
	Rows      []IndexRow
	RowsJSON  template.JS
	Total     int
	Functions []FuncMeta
	Categories []string
}

func (d *Data) BuildIndex() IndexPageData {
	var rows []IndexRow
	for _, key := range d.Names {
		b := d.Binaries[key]
		var labels []string
		for _, fk := range b.Functions() {
			labels = append(labels, functionMetaByKey[fk].Label)
		}
		rows = append(rows, IndexRow{
			Name:      b.Name,
			URL:       "/lolbas/" + b.URLPath(),
			Category:  b.Category,
			Functions: labels,
			Mitre:     b.MitreIDs(),
		})
	}
	raw, _ := json.Marshal(rows)

	return IndexPageData{
		Rows:       rows,
		RowsJSON:   template.JS(raw),
		Total:      len(rows),
		Functions:  functionMeta,
		Categories: categoryOrder,
	}
}

// --- Detail page ---

type DetectionItem struct {
	Key      string
	Value    string
	IsLink   bool
}

type CommandView struct {
	DescriptionHTML template.HTML
	CodeHTML        template.HTML
	Usecase         string
	Privileges      string
	MitreID         string
	OperatingSystem string
	Tags            []string
}

type CategorySection struct {
	Key      string
	Label    string
	Commands []CommandView
}

type DetailPageData struct {
	Name            string
	DescriptionHTML template.HTML
	Aliases         []string
	Author          string
	Created         string
	Paths           []string
	CodeSamples     []string
	Detections      []DetectionItem
	Resources       []string
	Acknowledgement []Acknowledgement
	Sections        []CategorySection
	Category        string
	Slug            string
	MitreIDs        []string
}

func codeBlock(code string) template.HTML {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	return template.HTML(`<pre class="lb-code"><code>` + html.EscapeString(code) + `</code></pre>`)
}

func (d *Data) BuildDetail(category, slug string) (DetailPageData, bool) {
	b, ok := d.Binaries[category+"/"+slug]
	if !ok {
		return DetailPageData{}, false
	}

	page := DetailPageData{
		Name:            b.Name,
		DescriptionHTML: md(b.Description),
		Author:          b.Author,
		Created:         b.Created,
		Category:        b.Category,
		Slug:            b.Slug,
		MitreIDs:        b.MitreIDs(),
	}
	for _, a := range b.Aliases {
		if a.Alias != "" {
			page.Aliases = append(page.Aliases, a.Alias)
		}
	}
	for _, p := range b.FullPath {
		if p.Path != "" {
			page.Paths = append(page.Paths, p.Path)
		}
	}
	for _, c := range b.CodeSample {
		if c.Code != "" {
			page.CodeSamples = append(page.CodeSamples, c.Code)
		}
	}
	for _, r := range b.Resources {
		if r.Link != "" {
			page.Resources = append(page.Resources, r.Link)
		}
	}
	page.Acknowledgement = b.Acknowledgement

	for _, det := range b.Detection {
		for k, v := range det {
			if v == "" {
				continue
			}
			page.Detections = append(page.Detections, DetectionItem{
				Key:    k,
				Value:  v,
				IsLink: k != "IOC" && (strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")),
			})
		}
	}

	for _, group := range b.CommandsByCategory() {
		label := group.Key
		if fm, ok := functionMetaByKey[group.Key]; ok {
			label = fm.Label
		}
		section := CategorySection{Key: group.Key, Label: label}
		for _, c := range group.Commands {
			var tags []string
			for _, tagMap := range c.Tags {
				for k, v := range tagMap {
					tags = append(tags, k+": "+v)
				}
			}
			sort.Strings(tags)
			section.Commands = append(section.Commands, CommandView{
				DescriptionHTML: md(c.Description),
				CodeHTML:        codeBlock(c.Command),
				Usecase:         c.Usecase,
				Privileges:      c.Privileges,
				MitreID:         c.MitreID,
				OperatingSystem: c.OperatingSystem,
				Tags:            tags,
			})
		}
		page.Sections = append(page.Sections, section)
	}

	return page, true
}
