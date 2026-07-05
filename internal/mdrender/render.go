package mdrender

import (
	"bytes"
	"regexp"
	"strings"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Footnote,
		highlighting.NewHighlighting(
			highlighting.WithStyle("onedark"),
			highlighting.WithGuessLanguage(true),
		),
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		parser.WithASTTransformers(
			util.Prioritized(&mdLinkTransformer{}, 999),
		),
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
		html.WithXHTML(),
	),
)

// mdBookDirective matches {{#include ...}}, {{#playground ...}}, etc.
// Also handles the edge case where a heading line is entirely a directive: "# {{#include ...}}"
var mdBookDirective = regexp.MustCompile(`(?m)(^#{1,6}\s+)?{{#\w+[^}]*}}`)

// stripMdBookDirectives removes mdBook-specific template directives before rendering.
// Lines that consist only of a directive (or "# {{#include ...}}") are removed entirely.
func stripMdBookDirectives(src []byte) []byte {
	result := mdBookDirective.ReplaceAllFunc(src, func(match []byte) []byte {
		s := strings.TrimSpace(string(match))
		// If the entire match is a heading-with-directive ("# {{...}}"), drop the whole line
		// If it's a bare directive, drop it
		_ = s
		return nil
	})
	// Collapse runs of blank lines left by stripped directives
	multiBlank := regexp.MustCompile(`\n{3,}`)
	return multiBlank.ReplaceAll(result, []byte("\n\n"))
}

// Render converts markdown bytes to an HTML string.
func Render(src []byte) (string, error) {
	src = stripMdBookDirectives(src)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// PageTitle extracts the first H1 heading from markdown source.
func PageTitle(src []byte) string {
	lines := strings.SplitN(string(src), "\n", 10)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}

// mdLinkTransformer rewrites *.md links to strip the extension.
type mdLinkTransformer struct{}

func (t *mdLinkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok {
			dest := string(link.Destination)
			if !isExternal(dest) && strings.HasSuffix(dest, ".md") {
				link.Destination = []byte(strings.TrimSuffix(dest, ".md"))
			}
		}
		return ast.WalkContinue, nil
	})
}

func isExternal(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
