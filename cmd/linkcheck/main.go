package main

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	attributePattern     = regexp.MustCompile(`(?is)\b(?:href|src)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^[:space:]"'=<>]+))`)
	headingPattern       = regexp.MustCompile(`(?is)<h1\b[^>]*>(.*?)</h1>`)
	tagPattern           = regexp.MustCompile(`(?is)<[^>]+>`)
	documentCountPattern = regexp.MustCompile(`(?i)\bdata-published-documents\s*=\s*"?([0-9]+)`)
	categoryCountPattern = regexp.MustCompile(`(?i)\bdata-published-categories\s*=\s*"?([0-9]+)`)
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "用法: linkcheck <public-directory>")
		os.Exit(2)
	}
	errorsFound, err := check(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "链接检查失败: %v\n", err)
		os.Exit(1)
	}
	if len(errorsFound) != 0 {
		for _, problem := range errorsFound {
			fmt.Fprintln(os.Stderr, problem)
		}
		os.Exit(1)
	}
	fmt.Println("内部链接检查通过")
}

func check(root string) ([]string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootInfo, err := os.Stat(absoluteRoot)
	if err != nil {
		return nil, err
	}
	if !rootInfo.IsDir() {
		return nil, errors.New("public path is not a directory")
	}

	problems := make(map[string]struct{})
	err = filepath.WalkDir(absoluteRoot, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("generated site contains symlink %q", filename)
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(filename)) != ".html" {
			return nil
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		relativeFile, _ := filepath.Rel(absoluteRoot, filename)
		relativeFile = filepath.ToSlash(relativeFile)
		checkPresentation(relativeFile, data, problems)
		for _, match := range attributePattern.FindAllSubmatch(data, -1) {
			if len(match) < 4 {
				continue
			}
			raw := ""
			for _, candidate := range match[1:] {
				if len(candidate) != 0 {
					raw = string(candidate)
					break
				}
			}
			raw = html.UnescapeString(strings.TrimSpace(raw))
			target, internal, err := resolve(absoluteRoot, filename, raw)
			if err != nil {
				problems[fmt.Sprintf("%s: 无效内部链接", relativeFile)] = struct{}{}
				continue
			}
			if !internal {
				continue
			}
			_, found := existingTarget(target)
			if found {
				continue
			}
			relativeTarget, _ := filepath.Rel(absoluteRoot, target)
			problems[fmt.Sprintf("%s: 缺少内部目标 %s", relativeFile, filepath.ToSlash(relativeTarget))] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(problems))
	for problem := range problems {
		result = append(result, problem)
	}
	sort.Strings(result)
	return result, nil
}

func checkPresentation(relativeFile string, data []byte, problems map[string]struct{}) {
	if relativeFile == "index.html" {
		_, documentsOK := positiveAttribute(data, documentCountPattern)
		_, categoriesOK := positiveAttribute(data, categoryCountPattern)
		featured := bytes.Count(data, []byte("hextra-feature-card"))
		if !bytes.Contains(data, []byte("data-knowledge-stats")) || !documentsOK || !categoriesOK || featured != 3 {
			problems["index.html: 知识导航首页标记不完整"] = struct{}{}
		}
	}
	if !strings.HasPrefix(relativeFile, "docs/") || filepath.Base(relativeFile) != "index.html" {
		return
	}
	headings := headingPattern.FindAllSubmatch(data, 2)
	if len(headings) < 2 {
		return
	}
	first := normalizedHeading(headings[0][1])
	second := normalizedHeading(headings[1][1])
	if first != "" && first == second {
		problems[fmt.Sprintf("%s: 连续重复页面标题", relativeFile)] = struct{}{}
	}
}

func positiveAttribute(data []byte, pattern *regexp.Regexp) (int, bool) {
	match := pattern.FindSubmatch(data)
	if len(match) != 2 {
		return 0, false
	}
	value, err := strconv.Atoi(string(match[1]))
	return value, err == nil && value > 0
}

func normalizedHeading(value []byte) string {
	plain := tagPattern.ReplaceAll(value, nil)
	return strings.Join(strings.Fields(html.UnescapeString(string(plain))), " ")
}

func resolve(root, page, raw string) (string, bool, error) {
	if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "//") {
		return "", false, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", true, err
	}
	if parsed.IsAbs() {
		if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
			return "", false, nil
		}
		if !strings.EqualFold(parsed.Hostname(), "4eversivan.github.io") {
			return "", false, nil
		}
	}
	if parsed.Path == "" {
		return "", false, nil
	}
	decoded, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return "", true, err
	}
	var target string
	if strings.HasPrefix(decoded, "/") {
		target = filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(decoded, "/")))
	} else {
		target = filepath.Join(filepath.Dir(page), filepath.FromSlash(decoded))
	}
	target = filepath.Clean(target)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", true, errors.New("link escapes public root")
	}
	return target, true, nil
}

func existingTarget(target string) (string, bool) {
	info, err := os.Stat(target)
	if err == nil {
		if info.IsDir() {
			candidate := filepath.Join(target, "index.html")
			info, err = os.Stat(candidate)
			return candidate, err == nil && info.Mode().IsRegular()
		}
		return target, info.Mode().IsRegular()
	}
	if info, candidateErr := os.Stat(filepath.Join(target, "index.html")); candidateErr == nil && info.Mode().IsRegular() {
		return filepath.Join(target, "index.html"), true
	}
	if info, candidateErr := os.Stat(target + ".html"); candidateErr == nil && info.Mode().IsRegular() {
		return target + ".html", true
	}
	return "", false
}
