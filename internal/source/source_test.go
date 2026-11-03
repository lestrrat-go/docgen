package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/docgen/internal/source"
	"github.com/stretchr/testify/require"
)

func TestNewGitHub(t *testing.T) {
	t.Run("repo root with git dir", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
		l, err := source.NewGitHub(root, "github.com/o/r/v3", "v3.0.0")
		require.NoError(t, err)
		require.Equal(t, "https://github.com/o/r/blob/v3.0.0/a.go#L3", l.URL("a.go", 3))
	})
	t.Run("subdirectory with git file", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: x\n"), 0o644))
		sub := filepath.Join(root, "sub")
		require.NoError(t, os.Mkdir(sub, 0o755))
		l, err := source.NewGitHub(sub, "github.com/o/r", "v1.0.0")
		require.NoError(t, err)
		require.Equal(t, "https://github.com/o/r/blob/v1.0.0/sub/a.go#L1", l.URL("a.go", 1))
	})
	t.Run("ref with slash", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
		l, err := source.NewGitHub(root, "github.com/o/r", "release/1")
		require.NoError(t, err)
		require.Equal(t, "https://github.com/o/r/blob/release/1/a.go#L1", l.URL("a.go", 1))
	})
	t.Run("not github", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
		_, err := source.NewGitHub(root, "example.com/x", "v1")
		require.ErrorContains(t, err, "--source-base-url")
	})
	t.Run("too few elements", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
		_, err := source.NewGitHub(root, "github.com/o", "v1")
		require.ErrorContains(t, err, "--source-base-url")
	})
	t.Run("empty ref", func(t *testing.T) {
		_, err := source.NewGitHub(t.TempDir(), "github.com/o/r", "")
		require.Error(t, err)
	})
	t.Run("no git", func(t *testing.T) {
		dir := t.TempDir()
		for p := dir; ; p = filepath.Dir(p) {
			if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
				t.Skip("a parent of the temp dir contains .git")
			}
			if filepath.Dir(p) == p {
				break
			}
		}
		_, err := source.NewGitHub(dir, "github.com/o/r", "v1")
		require.ErrorContains(t, err, "no .git found")
	})
}

func TestNewBase(t *testing.T) {
	l := source.NewBase("https://h/b/")
	require.Equal(t, "https://h/b/a%20b.go#L7", l.URL("a b.go", 7))
	require.Equal(t, "https://h/b/d/e.go#L1", l.URL("d/e.go", 1))
}
