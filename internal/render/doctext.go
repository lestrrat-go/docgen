package render

import (
	"go/doc/comment"
	"go/token"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/lestrrat-go/docgen/internal/load"
	"github.com/lestrrat-go/docgen/internal/markdown"
)

// docText parses text with p.Doc.Parser(), rewrites links, and prints Markdown.
// self is the name of the type whose doc is rendered, or "".
// It returns the Markdown and whether the doc carries a "Deprecated: " paragraph.
// p and r MUST NOT be nil.
func docText(p *load.Package, r *link.Resolver, text, self string, headingLevel int) ([]byte, bool) {
	parser := p.Doc.Parser()
	d := parser.Parse(text)
	w := &linker{p: p, r: r, self: self, parser: parser}
	for _, b := range d.Content {
		switch b := b.(type) {
		case *comment.Paragraph:
			b.Text = w.rewrite(b.Text, true)
		case *comment.Heading:
			b.Text = w.rewrite(b.Text, false)
		case *comment.List:
			for _, item := range b.Items {
				for _, blk := range item.Content {
					if para, ok := blk.(*comment.Paragraph); ok {
						para.Text = w.rewrite(para.Text, true)
					}
				}
			}
		}
	}
	return markdown.Print(d, headingLevel), markdown.IsDeprecated(d)
}

// linker holds the inputs of one docText call. It lives only for that call.
type linker struct {
	p      *load.Package
	r      *link.Resolver
	self   string
	parser *comment.Parser
}

// rewrite resolves doc links in in and, when auto is set, links type names found in plain text.
func (w *linker) rewrite(in []comment.Text, auto bool) []comment.Text {
	var out []comment.Text
	for _, t := range in {
		switch t := t.(type) {
		case comment.Plain:
			if auto {
				out = append(out, w.autoLink(string(t))...)
				continue
			}
			out = append(out, t)
		case *comment.DocLink:
			if url := w.r.DocLinkURL(w.p.ImportPath, t); url != "" {
				out = append(out, &comment.Link{Text: t.Text, URL: url})
				continue
			}
			// Text inlined from an unresolved link is NOT passed through the auto-link scanner.
			out = append(out, t.Text...)
		default:
			out = append(out, t)
		}
	}
	return mergePlain(out)
}

func mergePlain(in []comment.Text) []comment.Text {
	var out []comment.Text
	for _, t := range in {
		plain, ok := t.(comment.Plain)
		if ok && len(out) > 0 {
			if prev, ok := out[len(out)-1].(comment.Plain); ok {
				out[len(out)-1] = prev + plain
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// runEnd returns the end offset of the maximal identifier-rune run starting at i.
func runEnd(s string, i int) int {
	for i < len(s) {
		r, n := utf8.DecodeRuneInString(s[i:])
		if !isIdentRune(r) {
			break
		}
		i += n
	}
	return i
}

// autoLink splits s into plain text and links to type names.
func (w *linker) autoLink(s string) []comment.Text {
	var out []comment.Text
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			out = append(out, comment.Plain(plain.String()))
			plain.Reset()
		}
	}
	emitLink := func(text, url string) {
		flush()
		out = append(out, &comment.Link{Text: []comment.Text{comment.Plain(text)}, URL: url})
	}

	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		if !isIdentRune(r) {
			plain.WriteString(s[i : i+n])
			i += n
			continue
		}
		end := runEnd(s, i)
		run := s[i:end]
		prevDot := i > 0 && s[i-1] == '.'
		if prevDot || !isIdentStart(r) {
			plain.WriteString(run)
			i = end
			continue
		}

		// Qualified candidate: run "." Y with Y exported.
		if end+1 < len(s) && s[end] == '.' {
			yr, _ := utf8.DecodeRuneInString(s[end+1:])
			if isIdentStart(yr) {
				yEnd := runEnd(s, end+1)
				y := s[end+1 : yEnd]
				if token.IsExported(y) {
					if pkg := w.lookupPkg(run); pkg != nil {
						if url := w.typeURL(pkg, y); url != "" {
							emitLink(run+"."+y, url)
						} else {
							plain.WriteString(run + "." + y)
						}
						i = yEnd
						continue
					}
					// Unresolved: X is an unqualified candidate and ".Y" stays plain.
					w.unqualified(run, emitLink, &plain)
					plain.WriteString("." + y)
					i = yEnd
					continue
				}
			}
		}

		w.unqualified(run, emitLink, &plain)
		i = end
	}
	flush()
	return out
}

func (w *linker) unqualified(name string, emitLink func(text, url string), plain *strings.Builder) {
	if token.IsExported(name) && name != w.self {
		if url := w.typeURL(w.p.Types, name); url != "" {
			emitLink(name, url)
			return
		}
	}
	plain.WriteString(name)
}

// lookupPkg resolves a package qualifier to a type-checked package, or nil.
func (w *linker) lookupPkg(x string) *types.Package {
	if path, ok := w.parser.LookupPackage(x); ok {
		if path == "" {
			return w.p.Types
		}
		// An import path absent from Types.Imports() falls through to the module lookup.
		for _, imp := range w.p.Types.Imports() {
			if imp.Path() == path {
				return imp
			}
		}
	}
	if ip, ok := w.r.PackageByName(x); ok {
		if dp, ok := w.r.Package(ip); ok {
			return dp.Types
		}
	}
	return nil
}

// typeURL returns the link target for the package-level type name in pkg, or "".
func (w *linker) typeURL(pkg *types.Package, name string) string {
	obj, ok := pkg.Scope().Lookup(name).(*types.TypeName)
	if !ok {
		return ""
	}
	return w.r.TypeURL(w.p.ImportPath, obj)
}
