package render

import (
	"strings"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/lestrrat-go/docgen/internal/load"
	"github.com/lestrrat-go/docgen/internal/markdown"
	"github.com/lestrrat-go/docgen/internal/output"
)

// Index renders the module index file. m and r MUST NOT be nil.
func Index(m *load.Module, r *link.Resolver) []byte {
	parts := []string{
		output.Marker,
		"# " + markdown.EscapeText(m.Path),
		"## Packages",
	}
	if len(m.Packages) == 0 {
		parts = append(parts, "This module has no documented packages.")
		return []byte(strings.Join(parts, "\n\n") + "\n")
	}

	lines := make([]string, 0, len(m.Packages))
	for _, p := range m.Packages {
		file, _ := r.File(p.ImportPath)
		label := p.RelDir
		if label == "" {
			label = m.Path
		}
		line := "- " + markdown.Link(label, file)
		if markdown.IsDeprecated(p.Doc.Parser().Parse(p.Doc.Doc)) {
			line += deprecatedSuffix
		} else if syn := p.Doc.Synopsis(p.Doc.Doc); syn != "" {
			line += ": " + markdown.EscapeText(syn)
		}
		lines = append(lines, line)
	}
	parts = append(parts, strings.Join(lines, "\n"))
	return []byte(strings.Join(parts, "\n\n") + "\n")
}
