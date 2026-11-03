package link

import (
	"path"
	"strings"
)

// RelPath returns the slash path from the directory of fromFile to toFile.
// Both arguments are slash paths relative to the output directory.
// RelPath("shape/three.md", "shape.md") == "../shape.md"; RelPath("a.md", "b/c.md") == "b/c.md".
func RelPath(fromFile, toFile string) string {
	from := splitDir(path.Dir(fromFile))
	to := splitDir(path.Dir(toFile))

	common := 0
	for common < len(from) && common < len(to) && from[common] == to[common] {
		common++
	}

	var parts []string
	for range from[common:] {
		parts = append(parts, "..")
	}
	parts = append(parts, to[common:]...)
	parts = append(parts, path.Base(toFile))
	return strings.Join(parts, "/")
}

func splitDir(dir string) []string {
	if dir == "." || dir == "" {
		return nil
	}
	return strings.Split(dir, "/")
}
