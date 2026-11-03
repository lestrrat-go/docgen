package render

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/lestrrat-go/docgen/internal/load"
	"github.com/lestrrat-go/docgen/internal/markdown"
	"github.com/lestrrat-go/docgen/internal/output"
)

// SourceLinker turns a module-relative slash file path and a 1-based line into a URL.
type SourceLinker interface {
	URL(relFile string, line int) string
}

const deprecatedSuffix = " (deprecated)"

// pageRenderer holds the inputs of one Package call. It lives only for that call.
type pageRenderer struct {
	m   *load.Module
	r   *link.Resolver
	p   *load.Package
	src SourceLinker
}

// Package renders the file for p. src may be nil (no source links).
// m, r, and p MUST NOT be nil.
func Package(m *load.Module, r *link.Resolver, p *load.Package, src SourceLinker) ([]byte, error) {
	file, ok := r.File(p.ImportPath)
	if !ok {
		return nil, fmt.Errorf("package %s is not documented", p.ImportPath)
	}
	pr := &pageRenderer{m: m, r: r, p: p, src: src}
	d := p.Doc

	parts := []string{
		output.Marker,
		"# " + markdown.EscapeText("package "+p.Name),
		"`import \"" + p.ImportPath + "\"`",
		markdown.Link("Module index", link.RelPath(file, link.IndexFile)),
	}

	if text, _ := pr.docPart(d.Doc, "", 3); text != "" {
		parts = append(parts, section(link.AnchorOverview, "Overview"), text)
	}

	var index []string
	var body []string

	if len(d.Consts) > 0 {
		index = append(index, indexLine(0, "Constants", "#"+link.AnchorConstants))
		body = append(body, section(link.AnchorConstants, "Constants"))
		blocks, err := pr.valueBlocks(d.Consts, 3)
		if err != nil {
			return nil, err
		}
		body = append(body, blocks...)
	}
	if len(d.Vars) > 0 {
		index = append(index, indexLine(0, "Variables", "#"+link.AnchorVariables))
		body = append(body, section(link.AnchorVariables, "Variables"))
		blocks, err := pr.valueBlocks(d.Vars, 3)
		if err != nil {
			return nil, err
		}
		body = append(body, blocks...)
	}
	if len(d.Funcs) > 0 {
		body = append(body, section(link.AnchorFunctions, "Functions"))
		for _, f := range d.Funcs {
			lines, blocks, err := pr.funcBlock(f, "###", 4, 0)
			if err != nil {
				return nil, err
			}
			index = append(index, lines...)
			body = append(body, blocks...)
		}
	}
	if len(d.Types) > 0 {
		body = append(body, section(link.AnchorTypes, "Types"))
		for _, t := range d.Types {
			lines, blocks, err := pr.typeBlock(t)
			if err != nil {
				return nil, err
			}
			index = append(index, lines...)
			body = append(body, blocks...)
		}
	}

	if len(index) > 0 {
		parts = append(parts, section(link.AnchorIndex, "Index"), strings.Join(index, "\n"))
	}
	parts = append(parts, body...)
	return []byte(strings.Join(parts, "\n\n") + "\n"), nil
}

func section(anchor, title string) string {
	return anchorTag(anchor) + "\n## " + title
}

func anchorTag(name string) string {
	return `<a name="` + name + `"></a>`
}

func indexLine(depth int, text, url string) string {
	return strings.Repeat("  ", depth) + "- " + markdown.Link(text, url)
}

// docPart renders doc comment text without its trailing newline.
func (pr *pageRenderer) docPart(text, self string, level int) (string, bool) {
	out, deprecated := docText(pr.p, pr.r, text, self, level)
	return strings.TrimSuffix(string(out), "\n"), deprecated
}

func (pr *pageRenderer) sourcePart(pos token.Pos) (string, error) {
	if pr.src == nil {
		return "", nil
	}
	position := pr.p.Fset.Position(pos)
	rel, err := filepath.Rel(pr.m.Dir, position.Filename)
	if err != nil {
		return "", fmt.Errorf("source path of %s: %w", position.Filename, err)
	}
	return markdown.Link("source", pr.src.URL(filepath.ToSlash(rel), position.Line)), nil
}

func (pr *pageRenderer) valueBlocks(vs []*doc.Value, level int) ([]string, error) {
	var blocks []string
	for _, v := range vs {
		var parts []string
		var anchors strings.Builder
		for _, n := range v.Names {
			anchors.WriteString(anchorTag(n))
		}
		parts = append(parts, anchors.String())

		sig, err := signature(pr.p, pr.r, v.Decl)
		if err != nil {
			return nil, err
		}
		parts = append(parts, sig)

		srcPart, err := pr.sourcePart(v.Decl.Pos())
		if err != nil {
			return nil, err
		}
		if srcPart != "" {
			parts = append(parts, srcPart)
		}
		if text, _ := pr.docPart(v.Doc, "", level); text != "" {
			parts = append(parts, text)
		}
		blocks = append(blocks, strings.Join(parts, "\n\n"))
	}
	return blocks, nil
}

// symbolBlock renders anchor+heading, signature, source link, and doc text.
// It returns the block and the unescaped heading text (with the deprecation suffix, if any) for the index.
func (pr *pageRenderer) symbolBlock(anchor, headingPrefix, heading string, decl ast.Node, pos token.Pos,
	docStr, self string, level int) (string, string, error) {
	docMD, deprecated := pr.docPart(docStr, self, level)
	text := heading
	headingLine := markdown.EscapeText(heading)
	if deprecated {
		text += deprecatedSuffix
		headingLine += deprecatedSuffix
	}
	parts := []string{anchorTag(anchor) + "\n" + headingPrefix + " " + headingLine}

	sig, err := signature(pr.p, pr.r, decl)
	if err != nil {
		return "", "", err
	}
	parts = append(parts, sig)

	srcPart, err := pr.sourcePart(pos)
	if err != nil {
		return "", "", err
	}
	if srcPart != "" {
		parts = append(parts, srcPart)
	}
	if docMD != "" {
		parts = append(parts, docMD)
	}
	return strings.Join(parts, "\n\n"), text, nil
}

// funcBlock renders a plain func.
func (pr *pageRenderer) funcBlock(f *doc.Func, headingPrefix string, level, depth int) ([]string, []string, error) {
	return pr.funcOrMethod(f, "", headingPrefix, level, depth)
}

// funcOrMethod renders a func or method. recvType is the receiver type name for anchors ("" for plain funcs).
func (pr *pageRenderer) funcOrMethod(f *doc.Func, recvType, headingPrefix string, level, depth int) ([]string, []string, error) {
	heading := "func " + f.Name
	if f.Recv != "" {
		heading = "func (" + f.Recv + ") " + f.Name
	}
	anchor := link.Anchor(recvType, f.Name)
	block, text, err := pr.symbolBlock(anchor, headingPrefix, heading, f.Decl, f.Decl.Pos(), f.Doc, "", level)
	if err != nil {
		return nil, nil, err
	}
	return []string{indexLine(depth, text, "#"+anchor)}, []string{block}, nil
}

func (pr *pageRenderer) typeBlock(t *doc.Type) ([]string, []string, error) {
	pos := t.Decl.Pos()
	for _, spec := range t.Decl.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == t.Name {
			pos = ts.Name.Pos()
		}
	}
	block, text, err := pr.symbolBlock(t.Name, "###", "type "+t.Name, t.Decl, pos, t.Doc, t.Name, 4)
	if err != nil {
		return nil, nil, err
	}
	index := []string{indexLine(0, text, "#"+t.Name)}
	body := []string{block}

	consts, err := pr.valueBlocks(t.Consts, 5)
	if err != nil {
		return nil, nil, err
	}
	vars, err := pr.valueBlocks(t.Vars, 5)
	if err != nil {
		return nil, nil, err
	}
	body = append(body, consts...)
	body = append(body, vars...)

	for _, f := range t.Funcs {
		lines, blocks, err := pr.funcBlock(f, "####", 5, 1)
		if err != nil {
			return nil, nil, err
		}
		index = append(index, lines...)
		body = append(body, blocks...)
	}
	for _, m := range t.Methods {
		lines, blocks, err := pr.funcOrMethod(m, t.Name, "####", 5, 1)
		if err != nil {
			return nil, nil, err
		}
		index = append(index, lines...)
		body = append(body, blocks...)
	}
	return index, body, nil
}
