package markdown_test

import (
	"testing"

	"github.com/lestrrat-go/docgen/internal/markdown"
	"github.com/stretchr/testify/require"
)

func TestEscapeText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"backslash", `a\b`, `a\\b`},
		{"backtick", "a`b", "a\\`b"},
		{"star", "a*b", `a\*b`},
		{"underscore", "a_b", `a\_b`},
		{"brackets", "[a]", `\[a\]`},
		{"angle", "<a>", `\<a\>`},
		{"amp", "a&b", `a\&b`},
		{"tilde", "a~b", `a\~b`},
		{"pipe", "a|b", `a\|b`},
		{"bang", "a!b", `a\!b`},
		{"dollar", "a$b", `a\$b`},
		{"minus start", "-x", `\-x`},
		{"plus start", "+x", `\+x`},
		{"hash start", "#x", `\#x`},
		{"equals start", "=x", `\=x`},
		{"dotted number", "1999. x", `1999\. x`},
		{"paren number", "2) x", `2\) x`},
		{"digits only", "12x", "12x"},
		{"trim", "  a \t", "a"},
		{"multi line", " a \n -b\n\t2) c ", "a\n\\-b\n2\\) c"},
		{"middle dash", "a-b", "a-b"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, markdown.EscapeText(tc.in))
		})
	}
}

func TestLinkDest(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"#X", "<#X>"},
		{"a b", "<a%20b>"},
		{"a<b", "<a%3Cb>"},
		{"a>b", "<a%3Eb>"},
		{`a\b`, "<a%5Cb>"},
		{"a\x01b", "<a%01b>"},
		{"a\x7fb", "<a%7Fb>"},
		{"https://x/y?a=b&c=d", "<https://x/y?a=b&c=d>"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, markdown.LinkDest(tc.in), tc.in)
	}
}

func TestLink(t *testing.T) {
	require.Equal(t, `[a\*b](<#X>)`, markdown.Link("a*b", "#X"))
}
