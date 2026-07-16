package main

import (
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var attributePattern = regexp.MustCompile(`(?is)\b(?:href|src)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^[:space:]"'=<>]+))`)

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
				relative, _ := filepath.Rel(absoluteRoot, filename)
				problems[fmt.Sprintf("%s: 无效内部链接", filepath.ToSlash(relative))] = struct{}{}
				continue
			}
			if !internal || exists(target) {
				continue
			}
			relativeFile, _ := filepath.Rel(absoluteRoot, filename)
			relativeTarget, _ := filepath.Rel(absoluteRoot, target)
			problems[fmt.Sprintf("%s: 缺少内部目标 %s", filepath.ToSlash(relativeFile), filepath.ToSlash(relativeTarget))] = struct{}{}
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

func exists(target string) bool {
	info, err := os.Stat(target)
	if err == nil {
		if info.IsDir() {
			_, err = os.Stat(filepath.Join(target, "index.html"))
			return err == nil
		}
		return info.Mode().IsRegular()
	}
	if info, candidateErr := os.Stat(filepath.Join(target, "index.html")); candidateErr == nil && info.Mode().IsRegular() {
		return true
	}
	if info, candidateErr := os.Stat(target + ".html"); candidateErr == nil && info.Mode().IsRegular() {
		return true
	}
	return false
}
