package markdown_test

import (
	"go/doc/comment"
	"testing"

	"github.com/lestrrat-go/docgen/internal/markdown"
	"github.com/stretchr/testify/require"
)

func printText(text string, level int) string {
	d := (&comment.Parser{}).Parse(text)
	return string(markdown.Print(d, level))
}

func TestPrintParagraph(t *testing.T) {
	require.Equal(t, "first line\nsecond line\n", printText("first line\nsecond line\n", 3))
	require.Equal(t, "a\n\nb\n", printText("a\n\nb\n", 3))
}

func TestPrintHeading(t *testing.T) {
	text := "Intro.\n\n# Usage\n\nBody.\n"
	got := printText(text, 3)
	require.Equal(t, "Intro.\n\n### Usage\n\nBody.\n", got)
	require.NotContains(t, got, "{#")
	require.Equal(t, "Intro.\n\n###### Usage\n\nBody.\n", printText(text, 7))
	require.Equal(t, "Intro.\n\n# Usage\n\nBody.\n", printText(text, 0))
}

func TestPrintCode(t *testing.T) {
	require.Equal(t, "Intro.\n\n```\nx := 1\n```\n", printText("Intro.\n\n\tx := 1\n", 3))
	require.Equal(t, "````\na ``` b\n````\n", printText("Intro.\n\n\ta ``` b\n", 3)[len("Intro.\n\n"):])
}

func TestPrintList(t *testing.T) {
	require.Equal(t, "- one\n- two\n", printText("Intro:\n  - one\n  - two\n", 3)[len("Intro:\n\n"):])
	loose := printText("Intro:\n  - one\n\n  - two\n", 3)[len("Intro:\n\n"):]
	require.Equal(t, "- one\n\n- two\n", loose)
	require.Equal(t, "1. a\n2. b\n", printText("Intro:\n 1. a\n 2. b\n", 3)[len("Intro:\n\n"):])
	require.Equal(t, "- first\n  wrapped\n\n  second para\n", printText("Intro:\n  - first\n    wrapped\n\n    second para\n", 3)[len("Intro:\n\n"):])
}

func TestPrintDeprecated(t *testing.T) {
	d := (&comment.Parser{}).Parse("Deprecated: use X.\nSecond line.\n")
	require.Equal(t, "> **Deprecated:** use X.\n> Second line.\n", string(markdown.Print(d, 3)))
	require.True(t, markdown.IsDeprecated(d))
	require.False(t, markdown.IsDeprecated((&comment.Parser{}).Parse("Not Deprecated: here.\n")))
	require.True(t, markdown.IsDeprecated((&comment.Parser{}).Parse("Intro.\n\nDeprecated: later.\n")))
}

func TestPrintLinks(t *testing.T) {
	require.Equal(t, "see [https://go.dev/x](<https://go.dev/x>) now\n", printText("see https://go.dev/x now\n", 3))
	require.Equal(t, "the [Go](<https://go.dev>) site\n", printText("the [Go] site\n\n[Go]: https://go.dev\n", 3))
}

func TestPrintEscapes(t *testing.T) {
	require.Equal(t, "a \\*b\\* \\<nil\\> \\$HOME\n1999\\. x\n", printText("a *b* <nil> $HOME\n1999. x\n", 3))
}

func TestPrintLinkSpaces(t *testing.T) {
	d := &comment.Doc{Content: []comment.Block{&comment.Paragraph{Text: []comment.Text{
		comment.Plain("A "),
		&comment.Link{Text: []comment.Text{comment.Plain("Circle")}, URL: "#Circle"},
		comment.Plain(" wraps"),
	}}}}
	require.Equal(t, "A [Circle](<#Circle>) wraps\n", string(markdown.Print(d, 3)))
}

func TestPrintDocLinkPanics(t *testing.T) {
	d := (&comment.Parser{LookupSym: func(recv, name string) bool { return true }}).Parse("see [Foo]\n")
	require.PanicsWithValue(t, "markdown.Print: unresolved DocLink", func() { markdown.Print(d, 3) })
}

func TestPrintEmpty(t *testing.T) {
	require.Empty(t, markdown.Print((&comment.Parser{}).Parse(""), 3))
}
