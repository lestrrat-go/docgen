package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/docgen/internal/cli"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "modules", name))
	require.NoError(t, err)
	return dir
}

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	code := cli.Run(t.Context(), args, &so, &se)
	return code, so.String(), se.String()
}

func TestRunUsage(t *testing.T) {
	code, _, stderr := run(t, "--help")
	require.Equal(t, cli.ExitOK, code)
	require.Contains(t, stderr, "-module-dir")

	code, _, stderr = run(t, "--nope")
	require.Equal(t, cli.ExitError, code)
	require.Contains(t, stderr, "docgen: ")

	code, _, _ = run(t, "extra")
	require.Equal(t, cli.ExitError, code)

	code, _, stderr = run(t, "--source-ref", "v1", "--source-base-url", "https://x")
	require.Equal(t, cli.ExitError, code)
	require.Contains(t, stderr, "mutually exclusive")

	code, _, _ = run(t, "--goos", "")
	require.Equal(t, cli.ExitError, code)
}

func TestRunVersion(t *testing.T) {
	code, stdout, _ := run(t, "--version")
	require.Equal(t, cli.ExitOK, code)
	require.True(t, strings.HasPrefix(stdout, "docgen "))
}

func TestRunOutContainsModule(t *testing.T) {
	code, stdout, stderr := run(t, "--module-dir", fixture(t, "basic"), "--out", ".")
	require.Equal(t, cli.ExitError, code)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "must not contain the module directory")
}

func TestRunErrors(t *testing.T) {
	out := t.TempDir()
	code, stdout, stderr := run(t, "--module-dir", fixture(t, "broken"), "--out", out)
	require.Equal(t, cli.ExitError, code)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "docgen: ")

	code, _, stderr = run(t, "--module-dir", fixture(t, "collision"), "--out", out)
	require.Equal(t, cli.ExitError, code)
	require.Contains(t, stderr, "sub.md")

	code, _, stderr = run(t, "--module-dir", t.TempDir(), "--out", out)
	require.Equal(t, cli.ExitError, code)
	require.Contains(t, stderr, "no go.mod")
}

func TestRunGenerateAndCheck(t *testing.T) {
	module := fixture(t, "basic")
	out := t.TempDir()

	code, stdout, stderr := run(t, "--module-dir", module, "--out", out)
	require.Equal(t, cli.ExitOK, code, stderr)
	require.Empty(t, stdout)
	for _, f := range []string{"README.md", "basic.md", "shape.md", "shape/three.md"} {
		require.FileExists(t, filepath.Join(out, filepath.FromSlash(f)))
	}

	code, stdout, stderr = run(t, "--module-dir", module, "--out", out, "--check")
	require.Equal(t, cli.ExitOK, code, stderr)
	require.Empty(t, stdout)

	target := filepath.Join(out, "shape.md")
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(target, append(data, []byte("extra\n")...), 0o644))
	code, stdout, _ = run(t, "--module-dir", module, "--out", out, "--check")
	require.Equal(t, cli.ExitDiff, code)
	require.Equal(t, filepath.ToSlash(filepath.Join(out, "shape.md"))+"\n", stdout)

	// A rerun restores the file.
	code, _, _ = run(t, "--module-dir", module, "--out", out)
	require.Equal(t, cli.ExitOK, code)
	code, stdout, _ = run(t, "--module-dir", module, "--out", out, "--check")
	require.Equal(t, cli.ExitOK, code)
	require.Empty(t, stdout)
}

func TestRunSourceBaseURL(t *testing.T) {
	out := t.TempDir()
	code, _, stderr := run(t, "--module-dir", fixture(t, "basic"), "--out", out,
		"--source-base-url", "https://example.com/src")
	require.Equal(t, cli.ExitOK, code, stderr)
	data, err := os.ReadFile(filepath.Join(out, "shape.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "[source](<https://example.com/src/shape/shape.go#L")
}
