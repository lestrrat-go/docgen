package load_test

import (
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/docgen/internal/load"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "modules", name))
	require.NoError(t, err)
	return dir
}

func loadFixture(t *testing.T, name string, cfg load.Config) *load.Module {
	t.Helper()
	cfg.Dir = fixture(t, name)
	if cfg.GOOS == "" {
		cfg.GOOS = "linux"
	}
	if cfg.GOARCH == "" {
		cfg.GOARCH = "amd64"
	}
	m, err := load.Load(t.Context(), cfg)
	require.NoError(t, err)
	return m
}

func funcNames(p *load.Package) []string {
	var names []string
	for _, f := range p.Doc.Funcs {
		names = append(names, f.Name)
	}
	return names
}

func TestLoadBasic(t *testing.T) {
	m := loadFixture(t, "basic", load.Config{})
	require.Equal(t, "example.com/basic", m.Path)

	var paths, rels []string
	for _, p := range m.Packages {
		paths = append(paths, p.ImportPath)
		rels = append(rels, p.RelDir)
	}
	require.Equal(t, []string{"example.com/basic", "example.com/basic/shape", "example.com/basic/shape/three"}, paths)
	require.Equal(t, []string{"", "shape", "shape/three"}, rels)

	require.Contains(t, m.Std, "io")
	require.Contains(t, m.Std, "unsafe")

	shape := m.Packages[1]
	require.Equal(t, "shape", shape.Name)
	require.NotContains(t, funcNames(shape), "ExampleNewCircle")
	require.Contains(t, funcNames(shape), "Map")
}

func TestLoadBuildTags(t *testing.T) {
	tests := []struct {
		name string
		cfg  load.Config
		want []string
	}{
		{"default", load.Config{}, []string{"Common", "Linux"}},
		{"windows", load.Config{GOOS: "windows"}, []string{"Common", "Windows"}},
		{"foo", load.Config{Tags: []string{"foo"}}, []string{"Common", "Foo", "Linux"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := loadFixture(t, "buildtags", tc.cfg)
			require.Len(t, m.Packages, 1)
			require.Equal(t, tc.want, funcNames(m.Packages[0]))
		})
	}
}

func TestLoadBroken(t *testing.T) {
	_, err := load.Load(t.Context(), load.Config{Dir: fixture(t, "broken"), GOOS: "linux", GOARCH: "amd64"})
	require.Error(t, err)
	require.Contains(t, err.Error(), `cannot use "s"`)
}

func TestLoadNoGoMod(t *testing.T) {
	_, err := load.Load(t.Context(), load.Config{Dir: t.TempDir(), GOOS: "linux", GOARCH: "amd64"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no go.mod")
}
