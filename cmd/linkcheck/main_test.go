package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckInternalLinks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<a href="/docs/">docs</a><img src="https://example.com/a.png">`), 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<a href=/missing/><img src=/missing.png>`), 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<a href=/python/Python3.10安装>page</a>`), 0o644); err != nil {
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
