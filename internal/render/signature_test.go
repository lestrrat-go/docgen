package render_test

import (
	"go/ast"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/lestrrat-go/docgen/internal/load"
	"github.com/lestrrat-go/docgen/internal/render"
	"github.com/stretchr/testify/require"
)

type env struct {
	m *load.Module
	r *link.Resolver
}

func loadEnv(t *testing.T, name string) env {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "modules", name))
	require.NoError(t, err)
	m, err := load.Load(t.Context(), load.Config{Dir: dir, GOOS: "linux", GOARCH: "amd64"})
	require.NoError(t, err)
	r, err := link.New(m)
	require.NoError(t, err)
	return env{m: m, r: r}
}

func (e env) pkg(t *testing.T, importPath string) *load.Package {
	t.Helper()
	p, ok := e.r.Package(importPath)
	require.True(t, ok, importPath)
	return p
}

func TestSignature(t *testing.T) {
	e := loadEnv(t, "basic")
	shape := e.pkg(t, "example.com/basic/shape")
	basic := e.pkg(t, "example.com/basic")

	typeDecl := func(name string) ast.Node {
		for _, ty := range shape.Doc.Types {
			if ty.Name == name {
				return ty.Decl
			}
		}
		require.Failf(t, "type not found", "%s", name)
		return nil
	}

	t.Run("Circle", func(t *testing.T) {
		got, err := render.Signature(shape, e.r, typeDecl("Circle"))
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(got, "<pre>type Circle struct {"))
		require.True(t, strings.HasSuffix(got, "}</pre>"))
		require.Contains(t, got, `io.<a href="https://pkg.go.dev/io#Reader">Reader</a>`)
		require.Contains(t, got, `<a href="#Base">Base</a>`)
		require.Contains(t, got, "// contains filtered or unexported fields")
		require.NotContains(t, got, "hidden")
		require.NotContains(t, got, "Circle is a circle")
	})

	t.Run("List", func(t *testing.T) {
		got, err := render.Signature(shape, e.r, typeDecl("List"))
		require.NoError(t, err)
		require.NotContains(t, got, ">T</a>")
		require.Contains(t, got, `<a href="#Shape">Shape</a>`)
	})

	t.Run("Map", func(t *testing.T) {
		var got string
		for _, f := range shape.Doc.Funcs {
			if f.Name != "Map" {
				continue
			}
			var err error
			got, err = render.Signature(shape, e.r, f.Decl)
			require.NoError(t, err)
		}
		require.Contains(t, got, "<pre>func Map[")
		require.NotContains(t, got, ">T</a>")
	})

	t.Run("Kind consts", func(t *testing.T) {
		var got string
		for _, ty := range shape.Doc.Types {
			if ty.Name != "Kind" {
				continue
			}
			var err error
			got, err = render.Signature(shape, e.r, ty.Consts[0].Decl)
			require.NoError(t, err)
		}
		require.Contains(t, got, "iota")
		require.Contains(t, got, `<a href="#Kind">Kind</a>`)
	})

	t.Run("Arrow", func(t *testing.T) {
		var got string
		for _, v := range shape.Doc.Consts {
			if v.Names[0] != "Arrow" {
				continue
			}
			var err error
			got, err = render.Signature(shape, e.r, v.Decl)
			require.NoError(t, err)
		}
		require.Contains(t, got, "&#34;&lt;-&#34;")
	})

	t.Run("Do", func(t *testing.T) {
		var got string
		for _, f := range basic.Doc.Funcs {
			if f.Name != "Do" {
				continue
			}
			var err error
			got, err = render.Signature(basic, e.r, f.Decl)
			require.NoError(t, err)
		}
		require.Contains(t, got, "thirdparty.Thing")
		require.Contains(t, got, "secret.Secret")
		require.NotContains(t, got, "Thing</a>")
		require.NotContains(t, got, "Secret</a>")
		require.Contains(t, got, `<a href="shape.md#Circle">Circle</a>`)
		require.Contains(t, got, `<a href="https://pkg.go.dev/builtin#error">error</a>`)
	})
}
