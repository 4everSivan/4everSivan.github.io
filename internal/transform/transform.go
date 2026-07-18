package transform

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

var errInvalidPath = errors.New("invalid relative markdown path")

// Document converts a scanned Markdown document into deterministic Hugo content.
// It only operates on an in-memory copy and never writes to the source tree.
// modTime is the source file modification time recorded at discovery; it is
// emitted as front matter lastmod. A zero modTime omits the lastmod line.
func Document(relativePath string, source []byte, modTime time.Time) ([]byte, error) {
	if err := validateMarkdownPath(relativePath); err != nil {
		return nil, err
	}
	if !utf8.Valid(source) {
		return nil, errors.New("markdown is not valid UTF-8")
	}

	content := bytes.TrimPrefix(source, []byte("\xef\xbb\xbf"))
	if hasFrontMatter(content) {
		return nil, errors.New("existing front matter is not supported")
	}

	heading := firstHeading(content)
	title := heading.title
	body := content
	if title == "" {
		title = strings.TrimSuffix(path.Base(relativePath), ".md")
	} else {
		body = make([]byte, 0, len(content)-(heading.end-heading.start))
		body = append(body, content[:heading.start]...)
		body = append(body, content[heading.end:]...)
	}
	if title == "" {
		return nil, errors.New("document title is empty")
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	fmt.Fprintf(&out, "title: %s\n", quoteYAML(title))
	if !modTime.IsZero() {
		fmt.Fprintf(&out, "lastmod: %s\n", quoteYAML(modTime.Format(time.RFC3339)))
	}
	out.WriteString("---\n\n")
	out.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		out.WriteByte('\n')
	}
	return out.Bytes(), nil
}

// SectionIndex creates deterministic Hextra section metadata.
func SectionIndex(relativeDir string) ([]byte, error) {
	clean := path.Clean(relativeDir)
	if clean == "." || clean == "" {
		return []byte("---\ntitle: \"文档\"\n---\n\n"), nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return nil, errInvalidPath
	}
	title := path.Base(clean)
	return []byte(fmt.Sprintf("---\ntitle: %s\n---\n\n", quoteYAML(title))), nil
}

func validateMarkdownPath(relativePath string) error {
	clean := path.Clean(relativePath)
	if clean != relativePath || clean == "." || clean == ".." || path.IsAbs(clean) || strings.HasPrefix(clean, "../") || path.Ext(clean) != ".md" {
		return errInvalidPath
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return errInvalidPath
		}
	}
	return nil
}

func hasFrontMatter(content []byte) bool {
	trimmed := bytes.TrimLeft(content, "\r\n\t ")
	return bytes.HasPrefix(trimmed, []byte("---\n")) || bytes.HasPrefix(trimmed, []byte("---\r\n")) || bytes.HasPrefix(trimmed, []byte("+++\n")) || bytes.HasPrefix(trimmed, []byte("{\n"))
}

type headingMatch struct {
	title string
	start int
	end   int
}

func firstHeading(content []byte) headingMatch {
	inFence := false
	fence := ""
	for start := 0; start < len(content); {
		lineEnd, next := physicalLineEnd(content, start)
		trimmed := strings.TrimSpace(string(content[start:lineEnd]))
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence, fence = true, marker
			} else if marker == fence {
				inFence, fence = false, ""
			}
			start = next
			continue
		}
		if inFence || !strings.HasPrefix(trimmed, "# ") {
			start = next
			continue
		}
		title := headingTitle(trimmed)
		if title != "" {
			end := next
			if end < len(content) {
				blankEnd, blankNext := physicalLineEnd(content, end)
				if strings.TrimSpace(string(content[end:blankEnd])) == "" {
					end = blankNext
				}
			}
			return headingMatch{title: title, start: start, end: end}
		}
		start = next
	}
	return headingMatch{}
}

func physicalLineEnd(content []byte, start int) (end, next int) {
	relativeEnd := bytes.IndexByte(content[start:], '\n')
	if relativeEnd < 0 {
		return len(content), len(content)
	}
	end = start + relativeEnd
	return end, end + 1
}

func headingTitle(line string) string {
	title := strings.TrimSpace(strings.TrimPrefix(line, "# "))
	hashes := len(title)
	for hashes > 0 && title[hashes-1] == '#' {
		hashes--
	}
	if hashes < len(title) && hashes > 0 && (title[hashes-1] == ' ' || title[hashes-1] == '\t') {
		title = strings.TrimSpace(title[:hashes])
	}
	return title
}

func quoteYAML(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return "\"" + value + "\""
}
