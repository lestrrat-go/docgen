package link_test

import (
	"go/doc/comment"
	"go/types"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/lestrrat-go/docgen/internal/load"
	"github.com/stretchr/testify/require"
)

const (
	basicPath = "example.com/basic"
	shapePath = "example.com/basic/shape"
	threePath = "example.com/basic/shape/three"
)

func loadModule(t *testing.T, name string) *load.Module {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "modules", name))
	require.NoError(t, err)
	m, err := load.Load(t.Context(), load.Config{Dir: dir, GOOS: "linux", GOARCH: "amd64"})
	require.NoError(t, err)
	return m
}

func newBasic(t *testing.T) (*load.Module, *link.Resolver) {
	t.Helper()
	m := loadModule(t, "basic")
	r, err := link.New(m)
	require.NoError(t, err)
	return m, r
}

func typeName(t *testing.T, pkg *types.Package, name string) *types.TypeName {
	t.Helper()
	obj, ok := pkg.Scope().Lookup(name).(*types.TypeName)
	require.True(t, ok, name)
	return obj
}

func importOf(t *testing.T, pkg *types.Package, path string) *types.Package {
	t.Helper()
	for _, imp := range pkg.Imports() {
		if imp.Path() == path {
			return imp
		}
	}
	require.Failf(t, "import not found", "%s in %s", path, pkg.Path())
	return nil
}

func TestFiles(t *testing.T) {
	_, r := newBasic(t)
	for ip, want := range map[string]string{
		basicPath: "basic.md",
		shapePath: "shape.md",
		threePath: "shape/three.md",
	} {
		got, ok := r.File(ip)
		require.True(t, ok)
		require.Equal(t, want, got)
	}
	_, ok := r.File("example.com/basic/internal/secret")
	require.False(t, ok)

	ip, ok := r.PackageByName("shape")
	require.True(t, ok)
	require.Equal(t, shapePath, ip)
	_, ok = r.PackageByName("nothing")
	require.False(t, ok)
}

func TestTypeURL(t *testing.T) {
	m, r := newBasic(t)
	pkgs := map[string]*load.Package{}
	for _, p := range m.Packages {
		pkgs[p.ImportPath] = p
	}
	basic, shape := pkgs[basicPath].Types, pkgs[shapePath].Types

	reader := typeName(t, importOf(t, shape, "io"), "Reader")
	require.Equal(t, "https://pkg.go.dev/io#Reader", r.TypeURL(shapePath, reader))

	f64, ok := types.Universe.Lookup("float64").(*types.TypeName)
	require.True(t, ok)
	require.Equal(t, "https://pkg.go.dev/builtin#float64", r.TypeURL(shapePath, f64))

	circle := typeName(t, shape, "Circle")
	require.Equal(t, "../shape.md#Circle", r.TypeURL(threePath, circle))
	require.Equal(t, "#Circle", r.TypeURL(shapePath, circle))
	require.Equal(t, "shape.md#Circle", r.TypeURL(basicPath, circle))

	secret := typeName(t, importOf(t, basic, "example.com/basic/internal/secret"), "Secret")
	require.Empty(t, r.TypeURL(basicPath, secret))

	thing := typeName(t, importOf(t, basic, "example.com/thirdparty"), "Thing")
	require.Empty(t, r.TypeURL(basicPath, thing))

	// Unexported and type parameters are plain text.
	require.Empty(t, r.TypeURL(shapePath, typeName(t, shape, "inner")))
	list := typeName(t, shape, "List").Type().(*types.Named)
	require.Empty(t, r.TypeURL(shapePath, list.TypeParams().At(0).Obj()))
}

func TestDocLinkURL(t *testing.T) {
	_, r := newBasic(t)
	tests := []struct {
		name string
		from string
		l    comment.DocLink
		want string
	}{
		{"promoted method", shapePath, comment.DocLink{Name: "Hello", Recv: "Circle"}, "#Circle.Hello"},
		{"field", shapePath, comment.DocLink{Name: "R", Recv: "Circle"}, "#Circle"},
		{"interface method", shapePath, comment.DocLink{Name: "Area", Recv: "Shape"}, "#Shape"},
		{"unexported field", shapePath, comment.DocLink{Name: "hidden", Recv: "Circle"}, ""},
		{"internal package", basicPath, comment.DocLink{ImportPath: "example.com/basic/internal/secret", Name: "Secret"}, ""},
		{"std type", basicPath, comment.DocLink{ImportPath: "io", Name: "Reader"}, "https://pkg.go.dev/io#Reader"},
		{"package link", basicPath, comment.DocLink{ImportPath: shapePath}, "shape.md"},
		{"own package", shapePath, comment.DocLink{ImportPath: shapePath}, "#pkg-overview"},
		{"other package symbol", basicPath, comment.DocLink{ImportPath: shapePath, Name: "Circle"}, "shape.md#Circle"},
		{"own symbol", basicPath, comment.DocLink{Name: "Version"}, "#Version"},
		{"missing symbol", basicPath, comment.DocLink{Name: "nothing"}, ""},
		{"third party", basicPath, comment.DocLink{ImportPath: "example.com/thirdparty", Name: "Thing"}, "https://pkg.go.dev/example.com/thirdparty#Thing"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, r.DocLinkURL(tc.from, &tc.l))
		})
	}
}

func TestIsStd(t *testing.T) {
	_, r := newBasic(t)
	require.True(t, r.IsStd("io"))
	require.True(t, r.IsStd("net/http"))
	require.False(t, r.IsStd("internal/abi"))
	require.False(t, r.IsStd("example.com/basic"))
}

func TestCollision(t *testing.T) {
	_, err := link.New(loadModule(t, "collision"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "sub.md")
}

func TestAnchor(t *testing.T) {
	require.Equal(t, "Name", link.Anchor("", "Name"))
	require.Equal(t, "T.M", link.Anchor("T", "M"))
}
