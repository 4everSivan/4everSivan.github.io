package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckInternalLinks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), validHome(`<a href="/docs/">docs</a><img src="https://example.com/a.png">`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "index.html"), []byte(`<a href="../missing/">missing</a>`), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 {
		t.Fatalf("got %v, want one missing target", problems)
	}
}

func TestCheckFindsUnquotedMinifiedAttributes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), validHome(`<a href=/missing/><img src=/missing.png>`), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 2 {
		t.Fatalf("got %v, want two missing unquoted targets", problems)
	}
}

func TestCheckResolvesExtensionlessPageWhoseNameContainsDot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "python"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), validHome(`<a href=/python/Python3.10安装>page</a>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "python", "Python3.10安装.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("extensionless page link was not resolved: %v", problems)
	}
}

func TestResolveRejectsEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, internal, err := resolve(root, filepath.Join(root, "index.html"), "../outside"); err == nil || !internal {
		t.Fatalf("expected escaping internal link to fail, internal=%v err=%v", internal, err)
	}
}

func TestCheckRejectsRepeatedLeadingH1(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), validHome(""), 0o644); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "docs", "article")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte(`<h1>重复标题</h1><h1>重复标题</h1><h1>后续标题</h1>`), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsProblem(problems, "连续重复页面标题") {
		t.Fatalf("duplicate leading H1 was not rejected: %v", problems)
	}
}

func TestCheckAllowsDistinctLaterH1(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), validHome(""), 0o644); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "docs", "article")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte(`<h1>页面标题</h1><h1>后续标题</h1>`), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("distinct later H1 was rejected: %v", problems)
	}
}

func TestCheckRequiresKnowledgeHomeMarkers(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<h1>简陋首页</h1>`), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsProblem(problems, "知识导航首页标记不完整") {
		t.Fatalf("incomplete home was not rejected: %v", problems)
	}
}

func validHome(inner string) []byte {
	return []byte(`<h1>Sivan 学习文档</h1>` +
		`<div data-knowledge-stats data-published-documents=1 data-published-categories=1></div>` +
		strings.Repeat(`<a class="hextra-feature-card" href="/">分类</a>`, 6) + inner)
}

func containsProblem(problems []string, want string) bool {
	for _, problem := range problems {
		if strings.Contains(problem, want) {
			return true
		}
	}
	return false
}
