// Package markdown escapes text for GitHub-flavored Markdown and prints Go doc comments as Markdown.
package markdown

import (
	"fmt"
	"strings"
)

const inlineSet = "\\`*_[]<>&~|!$"

// EscapeText escapes s as standalone Markdown text (headings, index entries, synopses). It keeps '\n'.
// On each line it trims leading and trailing spaces and tabs, applies escapeInline, then the line-start rules.
func EscapeText(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = escapeLineStart(escapeInline(trimLine(line)))
	}
	return strings.Join(lines, "\n")
}

func trimLine(s string) string {
	return strings.Trim(s, " \t")
}

// escapeInline backslash-escapes the inline set only. It does not trim and does not apply
// line-start rules, so spaces next to links survive.
func escapeInline(s string) string {
	if !strings.ContainsAny(s, inlineSet) {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(inlineSet, s[i]) >= 0 {
			sb.WriteByte('\\')
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

// escapeLineStart applies the rules that depend on the first characters of a line.
func escapeLineStart(line string) string {
	if line == "" {
		return line
	}
	switch line[0] {
	case '+', '-', '#', '=':
		return "\\" + line
	}
	n := 0
	for n < len(line) && line[n] >= '0' && line[n] <= '9' {
		n++
	}
	if n > 0 && n < len(line) && (line[n] == '.' || line[n] == ')') {
		return line[:n] + "\\" + line[n:]
	}
	return line
}

// LinkDest returns url as an angle-bracket link destination: "<" + encoded + ">".
func LinkDest(url string) string {
	var sb strings.Builder
	sb.Grow(len(url) + 2)
	sb.WriteByte('<')
	for i := 0; i < len(url); i++ {
		c := url[i]
		switch {
		case c == ' ', c == '<', c == '>', c == '\\', c < 0x20, c == 0x7f:
			fmt.Fprintf(&sb, "%%%02X", c)
		default:
			sb.WriteByte(c)
		}
	}
	sb.WriteByte('>')
	return sb.String()
}

// Link returns "[" + EscapeText(text) + "](" + LinkDest(url) + ")".
func Link(text, url string) string {
	return "[" + EscapeText(text) + "](" + LinkDest(url) + ")"
}
