package cli_test

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/docgen/internal/cli"
	"github.com/lestrrat-go/docgen/internal/golden"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite golden files")

func goldenDir(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "golden", name))
	require.NoError(t, err)
	return dir
}

func generate(t *testing.T, module string, extra ...string) string {
	t.Helper()
	out := t.TempDir()
	args := append([]string{"--module-dir", fixture(t, module), "--out", out}, extra...)
	code, _, stderr := run(t, args...)
	require.Equal(t, cli.ExitOK, code, stderr)
	return out
}

func TestGolden(t *testing.T) {
	tests := []struct {
		name   string
		module string
		args   []string
	}{
		{"basic", "basic", nil},
		{"buildtags-default", "buildtags", nil},
		{"buildtags-windows", "buildtags", []string{"--goos", "windows"}},
		{"buildtags-foo", "buildtags", []string{"--tags", "foo"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := generate(t, tc.module, tc.args...)
			golden.CheckTree(t, goldenDir(t, tc.name), got, *update)
		})
	}
}

func readTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	}))
	return files
}

func TestGoldenSpotChecks(t *testing.T) {
	files := readTree(t, generate(t, "basic"))

	require.Contains(t, files["shape.md"], `<a href="https://pkg.go.dev/builtin#float64">float64</a>`)
	require.Contains(t, files["shape.md"], `io.<a href="https://pkg.go.dev/io#Reader">Reader</a>`)
	require.Contains(t, files["shape/three.md"], `<a href="../shape.md#Circle">Circle</a>`)
	require.Contains(t, files["shape/three.md"], `[Module index](<../README.md>)`)
	require.Contains(t, files["basic.md"], `[shape.Circle](<shape.md#Circle>)`)
	require.Contains(t, files["basic.md"], `thirdparty.Thing`)
	require.NotContains(t, files["basic.md"], `Thing</a>`)
	require.NotContains(t, files["basic.md"], `Secret</a>`)

	var threeLine string
	for line := range strings.SplitSeq(files["README.md"], "\n") {
		if strings.Contains(line, "shape/three") {
			threeLine = line
		}
	}
	require.Contains(t, threeLine, "(deprecated)")

	for name, content := range files {
		require.NotContains(t, content, "Example", name)
		require.NotContains(t, content, "```go", name)
	}
	for key := range files {
		require.NotContains(t, []string{"secret.md", "basictool.md", "examples.md"}, path.Base(key), key)
	}
	require.Len(t, files, 4)
}

func TestGoldenDeterminism(t *testing.T) {
	a := generate(t, "basic")
	b := generate(t, "basic")
	golden.CheckTree(t, a, b, false)
}
