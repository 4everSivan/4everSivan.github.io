package transform

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

var errInvalidPath = errors.New("invalid relative markdown path")

// Document converts a scanned Markdown document into deterministic Hugo content.
// It only operates on an in-memory copy and never writes to the source tree.
func Document(relativePath string, source []byte) ([]byte, error) {
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

	title := firstHeading(content)
	if title == "" {
		title = strings.TrimSuffix(path.Base(relativePath), ".md")
	}
	if title == "" {
		return nil, errors.New("document title is empty")
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "---\ntitle: %s\n---\n\n", quoteYAML(title))
	out.Write(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
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

func firstHeading(content []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	inFence := false
	fence := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence, fence = true, marker
			} else if marker == fence {
				inFence, fence = false, ""
			}
			continue
		}
		if inFence || !strings.HasPrefix(trimmed, "# ") {
			continue
		}
		title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		title = strings.TrimSpace(strings.TrimSuffix(title, "#"))
		if title != "" {
			return title
		}
	}
	return ""
}

func quoteYAML(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return "\"" + value + "\""
}
