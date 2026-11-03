package render_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/docgen/internal/output"
	"github.com/lestrrat-go/docgen/internal/render"
	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	e := loadEnv(t, "basic")
	got := string(render.Index(e.m, e.r))
	require.True(t, strings.HasPrefix(got, output.Marker+"\n\n# example.com/basic\n\n## Packages\n\n"))
	require.Contains(t, got, "- [example.com/basic](<basic.md>)")
	require.Contains(t, got, "- [shape/three](<shape/three.md>) (deprecated)")
	require.Contains(t, got, "- [shape](<shape.md>): Package shape defines shapes.")
	require.True(t, strings.HasSuffix(got, "\n"))
}
