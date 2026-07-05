package wiki

import (
	"bufio"
	"html/template"
	"os"
	"strings"
)

// NavNode is a node in the SUMMARY.md navigation tree.
type NavNode struct {
	Title    string
	Path     string   // URL path, empty for section headers
	Children []*NavNode
	IsHeader bool     // ## Section Header
}

// parseSummary reads a GitBook/mdBook SUMMARY.md and returns a nav tree.
func parseSummary(summaryPath, prefix string) []*NavNode {
	f, err := os.Open(summaryPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var roots []*NavNode
	var stack []*NavNode // stack tracks current depth by indent level

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Section headers: ## Heading
		if strings.HasPrefix(strings.TrimSpace(line), "## ") || strings.HasPrefix(strings.TrimSpace(line), "# ") {
			text := strings.TrimSpace(line)
			text = strings.TrimLeft(text, "# ")
			node := &NavNode{Title: text, IsHeader: true}
			roots = append(roots, node)
			stack = []*NavNode{node}
			continue
		}

		// List item: (spaces)* [Title](path)
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, "* ") {
			continue
		}

		indent := len(line) - len(trimmed)
		trimmed = trimmed[2:] // strip "- " or "* "

		title, path := parseMDLink(trimmed)
		if title == "" {
			continue
		}
		if cleanTitle, externalPath, ok := stripExternalMarker(title); ok {
			title, path = cleanTitle, externalPath
		}

		// Remove .md extension and clean path for URL
		urlPath := ""
		if path != "" {
			if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
				urlPath = path
			} else {
				urlPath = prefix + "/" + strings.TrimSuffix(path, ".md")
			}
		}

		node := &NavNode{Title: title, Path: urlPath}

		depth := indent / 2 // assume 2-space indent
		if depth == 0 {
			if len(stack) > 0 && stack[0].IsHeader {
				stack[0].Children = append(stack[0].Children, node)
			} else {
				roots = append(roots, node)
				stack = []*NavNode{node}
			}
		} else {
			// Find parent at depth-1
			for len(stack) > depth {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
			if len(stack) <= depth {
				stack = append(stack, node)
			} else {
				stack[depth] = node
			}
		}
	}
	return roots
}

// parseMDLink extracts title and path from "[Title](path)" markdown link syntax.
func parseMDLink(s string) (title, path string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return "", ""
	}
	end := strings.Index(s, "]")
	if end < 0 {
		return "", ""
	}
	title = s[1:end]
	rest := s[end+1:]
	if !strings.HasPrefix(rest, "(") {
		return title, ""
	}
	close := strings.Index(rest, ")")
	if close < 0 {
		return title, ""
	}
	path = rest[1:close]
	// Strip anchor from path
	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[:idx]
	}
	return title, path
}

// stripExternalMarker handles HackTricks' "Title$$external:target$$" SUMMARY.md
// syntax: their own build preprocesses it into a real link before rendering,
// but the raw SUMMARY.md (which is all we read) still has it embedded in the
// link title with an empty target in the parens. target may be an absolute
// URL or a relative path to another page in the same site.
func stripExternalMarker(title string) (cleanTitle, target string, ok bool) {
	const marker = "$$external:"
	idx := strings.Index(title, marker)
	if idx < 0 {
		return "", "", false
	}
	rest := title[idx+len(marker):]
	end := strings.Index(rest, "$$")
	if end < 0 {
		return "", "", false
	}
	return strings.TrimSpace(title[:idx]), rest[:end], true
}

// RenderNav generates the sidebar HTML for the navigation tree.
func RenderNav(nodes []*NavNode, currentPath string) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<ul class="nav-tree">`)
	renderNodes(&sb, nodes, currentPath)
	sb.WriteString(`</ul>`)
	return template.HTML(sb.String())
}

func renderNodes(sb *strings.Builder, nodes []*NavNode, currentPath string) {
	for _, n := range nodes {
		if n.IsHeader {
			sb.WriteString(`<li class="nav-section">`)
			sb.WriteString(`<span class="nav-section-title">`)
			sb.WriteString(template.HTMLEscapeString(n.Title))
			sb.WriteString(`</span>`)
			if len(n.Children) > 0 {
				active := containsPath(n.Children, currentPath)
				if active {
					sb.WriteString(`<ul class="nav-children open">`)
				} else {
					sb.WriteString(`<ul class="nav-children">`)
				}
				renderNodes(sb, n.Children, currentPath)
				sb.WriteString(`</ul>`)
			}
			sb.WriteString(`</li>`)
			continue
		}

		if len(n.Children) > 0 {
			active := n.Path == currentPath || containsPath(n.Children, currentPath)
			cls := "nav-item has-children"
			if active {
				cls += " open"
			}
			sb.WriteString(`<li class="` + cls + `">`)
			if n.Path != "" {
				linkCls := "nav-link"
				if n.Path == currentPath {
					linkCls += " active"
				}
				sb.WriteString(`<a class="` + linkCls + `" href="` + template.HTMLEscapeString(n.Path) + `">`)
				sb.WriteString(template.HTMLEscapeString(n.Title))
				sb.WriteString(`</a>`)
			} else {
				sb.WriteString(`<span class="nav-group-label">`)
				sb.WriteString(template.HTMLEscapeString(n.Title))
				sb.WriteString(`</span>`)
			}
			sb.WriteString(`<ul class="nav-children">`)
			renderNodes(sb, n.Children, currentPath)
			sb.WriteString(`</ul>`)
			sb.WriteString(`</li>`)
		} else {
			cls := "nav-item"
			linkCls := "nav-link"
			if n.Path == currentPath {
				cls += " active"
				linkCls += " active"
			}
			sb.WriteString(`<li class="` + cls + `">`)
			if n.Path != "" {
				sb.WriteString(`<a class="` + linkCls + `" href="` + template.HTMLEscapeString(n.Path) + `">`)
				sb.WriteString(template.HTMLEscapeString(n.Title))
				sb.WriteString(`</a>`)
			} else {
				sb.WriteString(`<span>` + template.HTMLEscapeString(n.Title) + `</span>`)
			}
			sb.WriteString(`</li>`)
		}
	}
}

func containsPath(nodes []*NavNode, path string) bool {
	for _, n := range nodes {
		if n.Path == path {
			return true
		}
		if containsPath(n.Children, path) {
			return true
		}
	}
	return false
}
