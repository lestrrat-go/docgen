package output_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/docgen/internal/output"
	"github.com/stretchr/testify/require"
)

func gen(path, body string) output.File {
	return output.File{Path: path, Content: []byte(output.Marker + "\n\n" + body + "\n")}
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
}

func TestDiffMissingDir(t *testing.T) {
	out := filepath.Join(t.TempDir(), "docs")
	plan, err := output.Diff(out, []output.File{gen("b.md", "b"), gen("a/c.md", "c")})
	require.NoError(t, err)
	require.Equal(t, []string{"a/c.md", "b.md"}, plan.Write)
	require.Empty(t, plan.Delete)
}

func TestDiffIdenticalAndChanged(t *testing.T) {
	out := t.TempDir()
	same, changed := gen("same.md", "x"), gen("changed.md", "new")
	write(t, out, "same.md", string(same.Content))
	write(t, out, "changed.md", output.Marker+"\n\nold\n")
	plan, err := output.Diff(out, []output.File{same, changed})
	require.NoError(t, err)
	require.Equal(t, []string{"changed.md"}, plan.Write)
	require.Equal(t, []string{"changed.md"}, plan.Paths())
}

func TestDiffRefusesUnmarked(t *testing.T) {
	out := t.TempDir()
	write(t, out, "README.md", "# mine\n")
	_, err := output.Diff(out, []output.File{gen("README.md", "x")})
	require.ErrorContains(t, err, "refusing to overwrite")
}

func TestApplyDeletesStale(t *testing.T) {
	out := t.TempDir()
	keep := gen("keep.md", "k")
	write(t, out, "keep.md", string(keep.Content))
	write(t, out, "gone/deep/stale.md", output.Marker+"\n\nold\n")
	write(t, out, "notes.md", "# human notes\n")
	write(t, out, "gone/other.txt", "text")

	plan, err := output.Diff(out, []output.File{keep})
	require.NoError(t, err)
	require.Equal(t, []string{"gone/deep/stale.md"}, plan.Delete)
	require.NoError(t, output.Apply(out, []output.File{keep}, plan))

	require.NoFileExists(t, filepath.Join(out, "gone/deep/stale.md"))
	require.NoDirExists(t, filepath.Join(out, "gone/deep"))
	require.FileExists(t, filepath.Join(out, "gone/other.txt"))
	require.FileExists(t, filepath.Join(out, "notes.md"))
}

func TestApplyRemovesEmptyParents(t *testing.T) {
	out := t.TempDir()
	write(t, out, "gone/deep/stale.md", output.Marker+"\n")
	plan, err := output.Diff(out, nil)
	require.NoError(t, err)
	require.NoError(t, output.Apply(out, nil, plan))
	require.NoDirExists(t, filepath.Join(out, "gone"))
	require.DirExists(t, out)
}

func TestApplyThenDiffEmpty(t *testing.T) {
	out := filepath.Join(t.TempDir(), "docs")
	files := []output.File{gen("a.md", "a"), gen("sub/b.md", "b")}
	plan, err := output.Diff(out, files)
	require.NoError(t, err)
	require.NoError(t, output.Apply(out, files, plan))

	plan, err = output.Diff(out, files)
	require.NoError(t, err)
	require.Empty(t, plan.Paths())

	tmps, err := filepath.Glob(filepath.Join(out, "*", ".docgen-*"))
	require.NoError(t, err)
	require.Empty(t, tmps)
	tmps, err = filepath.Glob(filepath.Join(out, ".docgen-*"))
	require.NoError(t, err)
	require.Empty(t, tmps)

	info, err := os.Stat(filepath.Join(out, "a.md"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}
