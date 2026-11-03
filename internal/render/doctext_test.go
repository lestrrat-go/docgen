package render_test

import (
	"testing"

	"github.com/lestrrat-go/docgen/internal/render"
	"github.com/stretchr/testify/require"
)

const (
	basicPath = "example.com/basic"
	shapePath = "example.com/basic/shape"
)

func TestDocTextBasic(t *testing.T) {
	e := loadEnv(t, "basic")
	basic := e.pkg(t, basicPath)

	got, deprecated := render.DocText(basic, e.r, basic.Doc.Doc, "", 3)
	s := string(got)
	require.False(t, deprecated)
	require.Contains(t, s, "[shape.Circle](<shape.md#Circle>)")
	require.Contains(t, s, "example.com/basic/internal/secret.Secret")
	require.NotContains(t, s, "[example.com/basic/internal/secret.Secret")
	require.NotContains(t, s, "[example.com/basic/internal/secret.Secret]")
	require.Contains(t, s, `\[nothing\]`)
	require.Contains(t, s, "[io.Reader](<https://pkg.go.dev/io#Reader>)")
	require.Contains(t, s, "[Version](<#Version>)")
	require.Contains(t, s, "### Usage")
	require.Contains(t, s, "- a reader")
	require.Contains(t, s, "```\nerr := basic.Do(r, c, t, s)\n```")
	require.Contains(t, s, "[https://example.com/docs](<https://example.com/docs>)")
	require.Contains(t, s, "[Go site](<https://go.dev>)")
}

func TestDocTextShape(t *testing.T) {
	e := loadEnv(t, "basic")
	shape := e.pkg(t, shapePath)

	got, deprecated := render.DocText(shape, e.r, shape.Doc.Doc, "", 3)
	s := string(got)
	require.False(t, deprecated)
	require.Contains(t, s, "[Circle](<#Circle>)")
	require.Contains(t, s, "[io.Reader](<https://pkg.go.dev/io#Reader>)")
	require.Contains(t, s, "bytes.Buffer")
	require.NotContains(t, s, "[bytes.Buffer]")
	require.Contains(t, s, "[basic.Config](<basic.md#Config>)")
	require.NotContains(t, s, "[basic.Version]")
	require.Contains(t, s, "basic.Version")
	require.Contains(t, s, `\$HOME`)
	require.Contains(t, s, `\<nil\>`)
	require.Contains(t, s, `\*bold\*`)
	require.Contains(t, s, `\&amp;`)
	require.Contains(t, s, `\~/path`)
	require.Contains(t, s, `a \| b`)
	require.Contains(t, s, "\n1999\\. ")
}

func TestDocTextTypes(t *testing.T) {
	e := loadEnv(t, "basic")
	shape := e.pkg(t, shapePath)
	docs := map[string]string{}
	for _, ty := range shape.Doc.Types {
		docs[ty.Name] = ty.Doc
	}

	got, deprecated := render.DocText(shape, e.r, docs["Circle"], "Circle", 4)
	require.False(t, deprecated)
	require.Equal(t, "Circle is a circle, not a square.\n", string(got))

	got, _ = render.DocText(shape, e.r, docs["Circle"], "", 4)
	require.Contains(t, string(got), "[Circle](<#Circle>)")

	got, deprecated = render.DocText(shape, e.r, docs["OldShape"], "OldShape", 4)
	require.True(t, deprecated)
	require.Contains(t, string(got), "> **Deprecated:** use [Circle](<#Circle>).")
}
