// Package golden compares generated file trees with golden trees in tests.
package golden

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const (
	maxLineBytes = 200
	maxReports   = 10
)

// CheckTree compares the sets of relative file paths and each file's bytes under wantDir and gotDir.
// Diagnostics are bounded: a mismatch reports the file path, the first differing line number,
// and both lines truncated to 200 bytes. With update, it replaces wantDir with gotDir's contents.
func CheckTree(t testing.TB, wantDir, gotDir string, update bool) {
	t.Helper()
	if update {
		if err := replaceTree(wantDir, gotDir); err != nil {
			t.Fatalf("update golden tree %s: %v", wantDir, err)
		}
		return
	}

	want, err := listFiles(wantDir)
	if err != nil {
		t.Fatalf("list %s: %v", wantDir, err)
	}
	got, err := listFiles(gotDir)
	if err != nil {
		t.Fatalf("list %s: %v", gotDir, err)
	}

	reports := 0
	report := func(format string, args ...any) {
		reports++
		if reports <= maxReports {
			t.Errorf(format, args...)
		}
	}
	for _, p := range want {
		if !slices.Contains(got, p) {
			report("missing file %s (present in %s)", p, wantDir)
		}
	}
	for _, p := range got {
		if !slices.Contains(want, p) {
			report("unexpected file %s (absent from %s)", p, wantDir)
		}
	}
	for _, p := range want {
		if !slices.Contains(got, p) {
			continue
		}
		wantData, err := os.ReadFile(filepath.Join(wantDir, filepath.FromSlash(p)))
		if err != nil {
			t.Fatalf("read golden %s: %v", p, err)
		}
		gotData, err := os.ReadFile(filepath.Join(gotDir, filepath.FromSlash(p)))
		if err != nil {
			t.Fatalf("read generated %s: %v", p, err)
		}
		if bytes.Equal(wantData, gotData) {
			continue
		}
		line, w, g := firstDiff(wantData, gotData)
		report("%s differs at line %d\nwant: %s\n got: %s", p, line, truncate(w), truncate(g))
	}
	if reports > maxReports {
		t.Errorf("... and %d more differences", reports-maxReports)
	}
}

// firstDiff returns the 1-based number of the first differing line and both lines.
func firstDiff(want, got []byte) (int, []byte, []byte) {
	wl := bytes.Split(want, []byte("\n"))
	gl := bytes.Split(got, []byte("\n"))
	for i := 0; i < max(len(wl), len(gl)); i++ {
		var w, g []byte
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if i >= len(wl) || i >= len(gl) || !bytes.Equal(w, g) {
			return i + 1, w, g
		}
	}
	return 0, nil, nil
}

func truncate(b []byte) string {
	if len(b) > maxLineBytes {
		return string(b[:maxLineBytes]) + "..."
	}
	return string(b)
}

// listFiles returns the sorted slash paths of regular files under dir. A missing dir yields no files.
func listFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	slices.Sort(files)
	return files, err
}

func replaceTree(wantDir, gotDir string) error {
	if err := os.RemoveAll(wantDir); err != nil {
		return err
	}
	files, err := listFiles(gotDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(wantDir, 0o755); err != nil {
		return err
	}
	for _, p := range files {
		data, err := os.ReadFile(filepath.Join(gotDir, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		target := filepath.Join(wantDir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
