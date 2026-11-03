package markdown

import (
	"fmt"
	"go/doc/comment"
	"strings"
)

const deprecatedPrefix = "Deprecated: "

// Print renders d as Markdown. Doc-comment headings use headingLevel (clamped to 1..6).
// d MUST contain no *comment.DocLink nodes; render.doctext rewrites them first.
// The result ends with exactly one '\n', or is empty when d has no content.
func Print(d *comment.Doc, headingLevel int) []byte {
	headingLevel = min(max(headingLevel, 1), 6)

	var blocks []string
	for _, b := range d.Content {
		s := block(b, headingLevel)
		if s == "" {
			continue
		}
		blocks = append(blocks, s)
	}
	if len(blocks) == 0 {
		return nil
	}
	return []byte(strings.Join(blocks, "\n\n") + "\n")
}

// IsDeprecated reports whether a top-level paragraph of d starts with "Deprecated: ".
func IsDeprecated(d *comment.Doc) bool {
	for _, b := range d.Content {
		p, ok := b.(*comment.Paragraph)
		if ok && isDeprecatedParagraph(p) {
			return true
		}
	}
	return false
}

func isDeprecatedParagraph(p *comment.Paragraph) bool {
	if len(p.Text) == 0 {
		return false
	}
	t, ok := p.Text[0].(comment.Plain)
	return ok && strings.HasPrefix(string(t), deprecatedPrefix)
}

// block renders one top-level block without a trailing newline.
func block(b comment.Block, headingLevel int) string {
	switch b := b.(type) {
	case *comment.Paragraph:
		lines := paragraphLines(b.Text)
		if len(lines) == 0 {
			return ""
		}
		if !isDeprecatedParagraph(b) {
			return strings.Join(lines, "\n")
		}
		lines[0] = "**Deprecated:**" + strings.TrimPrefix(lines[0], "Deprecated:")
		for i, l := range lines {
			lines[i] = "> " + l
		}
		return strings.Join(lines, "\n")
	case *comment.Heading:
		return strings.Repeat("#", headingLevel) + " " + strings.Trim(inline(b.Text), " \t")
	case *comment.Code:
		return code(b)
	case *comment.List:
		return list(b)
	}
	panic(fmt.Sprintf("markdown.Print: unknown block type %T", b))
}

func code(c *comment.Code) string {
	text := strings.TrimRight(c.Text, "\n")
	if text == "" {
		return ""
	}
	fence := strings.Repeat("`", max(3, longestRun(text, '`')+1))
	return fence + "\n" + text + "\n" + fence
}

func longestRun(s string, c byte) int {
	longest, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] != c {
			cur = 0
			continue
		}
		cur++
		longest = max(longest, cur)
	}
	return longest
}

func list(l *comment.List) string {
	var sb strings.Builder
	loose := l.BlankBetween()
	for i, item := range l.Items {
		if i > 0 {
			sb.WriteString("\n")
			if loose {
				sb.WriteString("\n")
			}
		}
		marker := "- "
		if item.Number != "" {
			marker = item.Number + ". "
		}
		indent := strings.Repeat(" ", len(marker))
		first := true
		for _, b := range item.Content {
			p, ok := b.(*comment.Paragraph)
			if !ok {
				panic(fmt.Sprintf("markdown.Print: unexpected list item block %T", b))
			}
			lines := paragraphLines(p.Text)
			if len(lines) == 0 {
				continue
			}
			if first {
				sb.WriteString(marker)
			} else {
				sb.WriteString("\n\n")
				sb.WriteString(indent)
			}
			first = false
			sb.WriteString(strings.Join(lines, "\n"+indent))
		}
		if first {
			sb.WriteString(strings.TrimRight(marker, " "))
		}
	}
	return sb.String()
}

// paragraphLines concatenates the inline output of every node, splits it on '\n', then trims each line at its
// ends and applies the line-start rules. Trimming never happens at node boundaries.
func paragraphLines(text []comment.Text) []string {
	s := inline(text)
	if strings.Trim(s, " \t\n") == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = escapeLineStart(trimLine(l))
	}
	return lines
}

func inline(text []comment.Text) string {
	var sb strings.Builder
	for _, t := range text {
		switch t := t.(type) {
		case comment.Plain:
			sb.WriteString(escapeInline(string(t)))
		case comment.Italic:
			sb.WriteString("*" + escapeInline(string(t)) + "*")
		case *comment.Link:
			sb.WriteString("[" + inline(t.Text) + "](" + LinkDest(t.URL) + ")")
		case *comment.DocLink:
			panic("markdown.Print: unresolved DocLink")
		default:
			panic(fmt.Sprintf("markdown.Print: unknown text type %T", t))
		}
	}
	return sb.String()
}
