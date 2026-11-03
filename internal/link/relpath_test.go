package link_test

import (
	"testing"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/stretchr/testify/require"
)

func TestRelPath(t *testing.T) {
	tests := []struct {
		from, to, want string
	}{
		{"a.md", "b.md", "b.md"},
		{"a.md", "b/c.md", "b/c.md"},
		{"shape/three.md", "shape.md", "../shape.md"},
		{"shape/three.md", "README.md", "../README.md"},
		{"a/b.md", "a/c.md", "c.md"},
		{"a/b.md", "c/d.md", "../c/d.md"},
		{"a/b/c.md", "a/d.md", "../d.md"},
		{"basic.md", "shape/three.md", "shape/three.md"},
		{"a/b.md", "a/b/c.md", "b/c.md"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, link.RelPath(tc.from, tc.to), "%s -> %s", tc.from, tc.to)
	}
}
