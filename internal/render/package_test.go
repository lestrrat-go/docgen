package render_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/docgen/internal/output"
	"github.com/lestrrat-go/docgen/internal/render"
	"github.com/stretchr/testify/require"
)

type stubSource struct{}

func (stubSource) URL(relFile string, line int) string {
	return fmt.Sprintf("https://x/%s#L%d", relFile, line)
}

func renderPkg(t *testing.T, e env, importPath string, src render.SourceLinker) string {
	t.Helper()
	out, err := render.Package(e.m, e.r, e.pkg(t, importPath), src)
	require.NoError(t, err)
	return string(out)
}

func TestPackageShape(t *testing.T) {
	e := loadEnv(t, "basic")
	got := renderPkg(t, e, shapePath, nil)

	require.True(t, strings.HasPrefix(got, output.Marker+"\n\n# package shape\n"))
	require.Contains(t, got, "[Module index](<README.md>)")
	require.Contains(t, got, "<a name=\"Circle.Scale\"></a>\n#### func (\\*Circle) Scale")
	require.Contains(t, got, `<a name="Circle.Hello"></a>`)
	require.Contains(t, got, `<a name="Small"></a><a name="Large"></a>`)
	require.Contains(t, got, "### type OldShape (deprecated)")
	require.Contains(t, got, "<a name=\"pkg-overview\"></a>\n## Overview")
	require.Contains(t, got, "<a name=\"pkg-index\"></a>\n## Index")
	require.NotContains(t, got, "Example")
	require.NotContains(t, got, "[source]")

	require.True(t, strings.HasSuffix(got, "\n"))
	require.False(t, strings.HasSuffix(got, "\n\n"))
	for i, line := range strings.Split(got, "\n") {
		require.Equal(t, strings.TrimRight(line, " \t"), line, "trailing space on line %d", i+1)
	}

	idx := strings.Index(got, "## Index")
	end := strings.Index(got, "<a name=\"pkg-constants\">")
	index := got[idx:end]
	circle := strings.Index(index, "[type Circle]")
	require.NotEqual(t, -1, circle)
	require.Greater(t, strings.Index(index, "[func (\\*Circle) Scale]"), circle)
	require.Contains(t, index, "  - [func NewCircle](<#NewCircle>)")
	require.Contains(t, index, "- [type OldShape (deprecated)](<#OldShape>)")
}

func TestPackageSourceLinks(t *testing.T) {
	e := loadEnv(t, "basic")
	got := renderPkg(t, e, shapePath, stubSource{})
	require.Contains(t, got, "[source](<https://x/shape/shape.go#L")
}
