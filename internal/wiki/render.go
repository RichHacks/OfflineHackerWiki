package wiki

import "hackerwiki/internal/mdrender"

func renderMarkdown(src []byte) (string, error) { return mdrender.Render(src) }
