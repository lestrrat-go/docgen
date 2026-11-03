// Package render produces the Markdown files for documented packages.
package render

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/scanner"
	"go/token"
	"go/types"
	"html"
	"strings"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/lestrrat-go/docgen/internal/load"
)

// signature prints node with go/printer and returns "<pre>" + linkified + "</pre>".
// p, r, and node MUST NOT be nil.
func signature(p *load.Package, r *link.Resolver, node ast.Node) (string, error) {
	node = prepareNode(node)

	// Do NOT wrap node in printer.CommentedNode: on Go 1.26 that dropped the
	// "contains filtered or unexported fields" line.
	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}
	if err := cfg.Fprint(&buf, p.Fset, node); err != nil {
		return "", fmt.Errorf("signature of %s: %w", nodeName(node), err)
	}

	// ast.Walk visits children in source order and go/printer prints each identifier once in that order.
	var idents []*ast.Ident
	ast.Inspect(node, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			idents = append(idents, id)
		}
		return true
	})

	src := buf.Bytes()
	mismatch := func(off int) error {
		return fmt.Errorf("signature of %s: identifier mismatch at offset %d", nodeName(node), off)
	}

	var out strings.Builder
	out.WriteString("<pre>")
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var scanErr error
	var s scanner.Scanner
	s.Init(file, src, func(pos token.Position, msg string) {
		if scanErr == nil {
			scanErr = fmt.Errorf("%s", msg)
		}
	}, 0)

	last := 0
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.IDENT {
			continue
		}
		off := file.Offset(pos)
		if len(idents) == 0 || idents[0].Name != lit {
			return "", mismatch(off)
		}
		id := idents[0]
		idents = idents[1:]
		out.WriteString(html.EscapeString(string(src[last:off])))
		out.WriteString(identHTML(p, r, id))
		last = off + len(lit)
	}
	if scanErr != nil || len(idents) > 0 {
		return "", mismatch(last)
	}
	out.WriteString(html.EscapeString(string(src[last:])))
	out.WriteString("</pre>")
	return out.String(), nil
}

// identHTML renders id as a link when it denotes a linkable type name.
// Uses is checked first, so an embedded field identifier resolves to its type name rather than the field.
func identHTML(p *load.Package, r *link.Resolver, id *ast.Ident) string {
	name := html.EscapeString(id.Name)
	obj, ok := p.Info.Uses[id].(*types.TypeName)
	if !ok {
		return name
	}
	url := r.TypeURL(p.ImportPath, obj)
	if url == "" {
		return name
	}
	return `<a href="` + html.EscapeString(url) + `">` + name + `</a>`
}

// prepareNode returns a copy of node without doc comments. It never mutates the original.
func prepareNode(node ast.Node) ast.Node {
	switch n := node.(type) {
	case *ast.FuncDecl:
		c := *n
		c.Doc = nil
		return &c
	case *ast.GenDecl:
		c := *n
		c.Doc = nil
		if len(c.Specs) == 1 {
			if ts, ok := c.Specs[0].(*ast.TypeSpec); ok {
				tc := *ts
				tc.Doc = nil
				c.Specs = []ast.Spec{&tc}
			}
		}
		return &c
	}
	return node
}

func nodeName(node ast.Node) string {
	switch n := node.(type) {
	case *ast.FuncDecl:
		return n.Name.Name
	case *ast.GenDecl:
		for _, spec := range n.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				return s.Name.Name
			case *ast.ValueSpec:
				return s.Names[0].Name
			}
		}
	}
	return fmt.Sprintf("%T", node)
}
