// Package source builds source-file URLs for generated documentation.
package source

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Linker builds source URLs. It is immutable after construction.
type Linker struct {
	base string // base URL without trailing slash
}

// NewGitHub derives https://github.com/<owner>/<repo>/blob/<ref>[/<subdir>] for the module.
func NewGitHub(moduleDir, modulePath, ref string) (*Linker, error) {
	if ref == "" {
		return nil, errors.New("empty source ref")
	}
	elems := strings.Split(modulePath, "/")
	if elems[0] != "github.com" || len(elems) < 3 {
		return nil, fmt.Errorf("cannot derive a GitHub URL from module path %s; use --source-base-url", modulePath)
	}
	owner, repo := elems[1], elems[2]

	root, ok := findRepoRoot(moduleDir)
	if !ok {
		return nil, fmt.Errorf("no .git found above %s; use --source-base-url", moduleDir)
	}
	subdir, err := filepath.Rel(root, moduleDir)
	if err != nil {
		return nil, fmt.Errorf("relative path of %s: %w", moduleDir, err)
	}
	subdir = filepath.ToSlash(subdir)
	if subdir == "." {
		subdir = ""
	}

	base := "https://github.com/" + owner + "/" + repo + "/blob/" + escapeSegments(ref)
	if subdir != "" {
		base += "/" + escapeSegments(subdir)
	}
	return &Linker{base: base}, nil
}

// NewBase uses baseURL (trailing '/' trimmed) as the prefix for module-relative files.
func NewBase(baseURL string) *Linker {
	return &Linker{base: strings.TrimRight(baseURL, "/")}
}

// URL returns base + "/" + relFile + "#L" + strconv.Itoa(line).
func (l *Linker) URL(relFile string, line int) string {
	return l.base + "/" + escapeSegments(relFile) + "#L" + strconv.Itoa(line)
}

// escapeSegments applies url.PathEscape to each '/'-separated segment of s.
func escapeSegments(s string) string {
	segs := strings.Split(s, "/")
	for i, seg := range segs {
		segs[i] = url.PathEscape(seg)
	}
	return strings.Join(segs, "/")
}

// findRepoRoot walks up from dir to the first directory containing a .git entry.
func findRepoRoot(dir string) (string, bool) {
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
