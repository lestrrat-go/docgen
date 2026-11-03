// Package link computes output file paths, anchors, and link targets for generated documentation.
package link

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/types"
	"slices"
	"strings"

	"github.com/lestrrat-go/docgen/internal/load"
)

// IndexFile is the module index file name.
const IndexFile = "README.md"

// Section anchors. The '-' cannot appear in a Go identifier, so these never collide with symbol anchors.
const (
	AnchorOverview  = "pkg-overview"
	AnchorIndex     = "pkg-index"
	AnchorConstants = "pkg-constants"
	AnchorVariables = "pkg-variables"
	AnchorFunctions = "pkg-functions"
	AnchorTypes     = "pkg-types"
)

// Anchor returns the anchor name for a symbol: name when recv == "", else recv + "." + name.
// recv MUST already be stripped of a leading '*' and of type parameters.
func Anchor(recv, name string) string {
	if recv == "" {
		return name
	}
	return recv + "." + name
}

// Resolver resolves output paths and link targets. It is immutable after New.
type Resolver struct {
	module  *load.Module
	pkgs    map[string]*load.Package
	files   map[string]string
	byName  map[string]string // package name -> import path; "" when ambiguous
	symbols map[string]map[string]string
}

// New computes output paths and symbol tables for m. m MUST NOT be nil.
// It returns an error naming both import paths when two packages map to the same output path.
func New(m *load.Module) (*Resolver, error) {
	if m == nil {
		return nil, errors.New("link.New: nil module")
	}
	r := &Resolver{
		module:  m,
		pkgs:    make(map[string]*load.Package, len(m.Packages)),
		files:   make(map[string]string, len(m.Packages)),
		byName:  map[string]string{},
		symbols: make(map[string]map[string]string, len(m.Packages)),
	}

	type producer struct{ path, label string }
	seen := map[string]producer{strings.ToLower(IndexFile): {IndexFile, "index file"}}
	for _, p := range m.Packages {
		file := p.RelDir + ".md"
		if p.RelDir == "" {
			file = p.Name + ".md"
		}
		key := strings.ToLower(file)
		if prev, ok := seen[key]; ok {
			labels := []string{prev.label, p.ImportPath}
			slices.Sort(labels)
			return nil, fmt.Errorf("output path collision: %s is produced by %s and %s", prev.path, labels[0], labels[1])
		}
		seen[key] = producer{file, p.ImportPath}

		r.pkgs[p.ImportPath] = p
		r.files[p.ImportPath] = file
		if _, dup := r.byName[p.Name]; dup {
			r.byName[p.Name] = ""
		} else {
			r.byName[p.Name] = p.ImportPath
		}
		r.symbols[p.ImportPath] = symbolTable(p.Doc)
	}
	return r, nil
}

func symbolTable(d *doc.Package) map[string]string {
	t := map[string]string{}
	values := func(vs []*doc.Value) {
		for _, v := range vs {
			for _, n := range v.Names {
				t[n] = n
			}
		}
	}
	funcs := func(fs []*doc.Func) {
		for _, f := range fs {
			t[f.Name] = f.Name
		}
	}
	values(d.Consts)
	values(d.Vars)
	funcs(d.Funcs)
	for _, ty := range d.Types {
		t[ty.Name] = ty.Name
		values(ty.Consts)
		values(ty.Vars)
		funcs(ty.Funcs)
		for _, m := range ty.Methods {
			key := Anchor(ty.Name, m.Name)
			t[key] = key
		}
		for _, member := range members(ty) {
			t[Anchor(ty.Name, member)] = ty.Name
		}
	}
	return t
}

// members returns the names of struct fields and interface methods of ty, read from the filtered declaration.
// Embedded fields (no Names) are excluded.
func members(ty *doc.Type) []string {
	if ty.Decl == nil {
		return nil
	}
	var names []string
	for _, spec := range ty.Decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		var fl *ast.FieldList
		switch x := ts.Type.(type) {
		case *ast.StructType:
			fl = x.Fields
		case *ast.InterfaceType:
			fl = x.Methods
		}
		if fl == nil {
			continue
		}
		for _, f := range fl.List {
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
		}
	}
	return names
}

// File returns the output path (slash, relative to the output directory) for a documented package.
func (r *Resolver) File(importPath string) (string, bool) {
	f, ok := r.files[importPath]
	return f, ok
}

// Package returns the documented package with importPath.
func (r *Resolver) Package(importPath string) (*load.Package, bool) {
	p, ok := r.pkgs[importPath]
	return p, ok
}

// PackageByName returns the import path of the only documented package named name.
// It returns false when zero or more than one documented package has that name.
func (r *Resolver) PackageByName(name string) (string, bool) {
	ip := r.byName[name]
	return ip, ip != ""
}

// IsStd reports whether importPath is a linkable standard library package.
func (r *Resolver) IsStd(importPath string) bool {
	if _, ok := r.module.Std[importPath]; !ok {
		return false
	}
	if strings.HasPrefix(importPath, "vendor/") {
		return false
	}
	return !slices.Contains(strings.Split(importPath, "/"), "internal")
}

// TypeURL returns the href for obj as seen from the file of package from, or "" for plain text.
func (r *Resolver) TypeURL(from string, obj *types.TypeName) string {
	if obj.Pkg() == nil {
		return "https://pkg.go.dev/builtin#" + obj.Name()
	}
	if _, ok := obj.Type().(*types.TypeParam); ok {
		return ""
	}
	if !obj.Exported() {
		return ""
	}
	if obj.Pkg().Scope().Lookup(obj.Name()) != obj {
		return ""
	}
	p := obj.Pkg().Path()
	if _, ok := r.pkgs[p]; ok {
		if p == from {
			return "#" + obj.Name()
		}
		return RelPath(r.files[from], r.files[p]) + "#" + obj.Name()
	}
	if r.IsStd(p) {
		return "https://pkg.go.dev/" + p + "#" + obj.Name()
	}
	return ""
}

// DocLinkURL returns the href for an explicit doc link inside package from, or "" for plain text.
func (r *Resolver) DocLinkURL(from string, l *comment.DocLink) string {
	ip := l.ImportPath
	if ip == "" {
		ip = from
	}
	if _, ok := r.pkgs[ip]; ok {
		prefix := ""
		if ip != from {
			prefix = RelPath(r.files[from], r.files[ip])
		}
		if l.Name == "" {
			if ip == from {
				return "#" + AnchorOverview
			}
			return prefix
		}
		anchor, ok := r.symbols[ip][Anchor(l.Recv, l.Name)]
		if !ok {
			return ""
		}
		return prefix + "#" + anchor
	}
	mp := r.module.Path
	if ip == mp || strings.HasPrefix(ip, mp+"/") {
		return ""
	}
	return l.DefaultURL("https://pkg.go.dev")
}
