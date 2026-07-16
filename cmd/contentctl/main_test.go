package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/4everSivan/4everSivan.github.io/internal/approval"
	"github.com/4everSivan/4everSivan.github.io/internal/report"
	"github.com/4everSivan/4everSivan.github.io/internal/snapshot"
)

func TestSyncAndVerifyClassifyEveryCandidate(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	sourceRoot := filepath.Join(t.TempDir(), "学习文档")
	if err := os.MkdirAll(filepath.Join(sourceRoot, "分类"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceRoot, "仅排除"), 0o755); err != nil {
		t.Fatal(err)
	}
	passedPath := filepath.Join(sourceRoot, "分类", "安全.md")
	excludedPath := filepath.Join(sourceRoot, "分类", "本地资源.md")
	excludedOnlyPath := filepath.Join(sourceRoot, "仅排除", "越界资源.md")
	if err := os.WriteFile(passedPath, []byte("# 安全文档\n\n![remote](https://example.com/a.png)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(excludedPath, []byte("# 本地资源\n\n![local](images/a.png)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(excludedOnlyPath, []byte("# 越界资源\n\n![local](../../outside.png)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := map[string]string{
		passedPath: fileHash(t, passedPath), excludedPath: fileHash(t, excludedPath), excludedOnlyPath: fileHash(t, excludedOnlyPath),
	}

	if err := os.WriteFile(filepath.Join(project, ".gitleaks.toml"), []byte("[extend]\nuseDefault = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "config", "versions.env"), []byte("GITLEAKS_VERSION=8.30.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeRuleFingerprintFiles(t, project)
	if err := approval.New().Save(filepath.Join(project, approval.ConfigPath)); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(project, "content", "docs", "旧分类", "旧文章.md")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "content", "_index.md"), []byte("# 站点入口\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeGitleaks := filepath.Join(project, ".local", "bin", "gitleaks")
	writeFakeGitleaks(t, fakeGitleaks)
	runtime := runtimeConfig{sourceRoot: sourceRoot, gitleaksSHA256: fileHash(t, fakeGitleaks)}

	var output bytes.Buffer
	args := []string{"sync", "--project-root", project}
	if err := runConfigured(t.Context(), args, &output, runtime); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "候选=3") || !strings.Contains(output.String(), "已同步=1") || !strings.Contains(output.String(), "已排除=2") {
		t.Fatalf("unexpected redacted summary: %s", output.String())
	}
	if _, err := os.Stat(filepath.Join(project, "content", "docs", "分类", "安全.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "content", "docs", "分类", "本地资源.md")); !os.IsNotExist(err) {
		t.Fatalf("excluded document entered snapshot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "content", "docs", "仅排除", "越界资源.md")); !os.IsNotExist(err) {
		t.Fatalf("excluded-only document entered snapshot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "content", "docs", "仅排除", "_index.md")); err != nil {
		t.Fatalf("excluded-only category was not preserved: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy project article survived atomic replacement: %v", err)
	}
	manifest, err := snapshot.LoadManifest(filepath.Join(project, "content", "docs", snapshot.ManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Documents) != 1 || manifest.Documents[0].Path != "分类/安全.md" {
		t.Fatalf("unexpected manifest: %+v", manifest.Documents)
	}
	if !hasManifestIndex(manifest, "仅排除/_index.md") {
		t.Fatalf("excluded-only category missing from manifest: %+v", manifest.GeneratedIndexes)
	}
	exclusions, err := report.Load(filepath.Join(project, report.LocalPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(exclusions.Documents) != 2 {
		t.Fatalf("unexpected exclusion report: %+v", exclusions.Documents)
	}
	for filename, hash := range before {
		if after := fileHash(t, filename); after != hash {
			t.Fatalf("source file changed: %s", filename)
		}
	}

	output.Reset()
	if err := runConfigured(t.Context(), []string{"verify", "--project-root", project}, &output, runtime); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "受控文档=1") {
		t.Fatalf("unexpected verify output: %s", output.String())
	}
	rulePath := filepath.Join(project, "internal", "scanner", "scanner.go")
	ruleBefore, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rulePath, append(ruleBefore, []byte("changed\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runConfigured(t.Context(), []string{"verify", "--project-root", project}, &bytes.Buffer{}, runtime); err == nil {
		t.Fatal("verify accepted a snapshot produced by different scanner source")
	}
	if err := os.WriteFile(rulePath, ruleBefore, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, extra := range []string{"extra.md", "extra.html"} {
		extraPath := filepath.Join(project, "content", extra)
		if err := os.WriteFile(extraPath, []byte("unexpected"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := runConfigured(t.Context(), []string{"verify", "--project-root", project}, &bytes.Buffer{}, runtime); err == nil {
			t.Fatalf("verify accepted uncontrolled content file %s", extra)
		}
		if err := os.Remove(extraPath); err != nil {
			t.Fatal(err)
		}
	}

	firstSnapshot, err := snapshot.HashTree(filepath.Join(project, "content", "docs"))
	if err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := runConfigured(t.Context(), args, &output, runtime); err != nil {
		t.Fatal(err)
	}
	secondSnapshot, err := snapshot.HashTree(filepath.Join(project, "content", "docs"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstSnapshot, secondSnapshot) {
		t.Fatal("two complete sync runs with identical inputs produced different snapshots")
	}

	reportPath := filepath.Join(project, report.LocalPath)
	if err := os.Remove(reportPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(reportPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passedPath, []byte("# 安全文档已更新\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runConfigured(t.Context(), args, &bytes.Buffer{}, runtime); err == nil {
		t.Fatal("sync unexpectedly succeeded when the local report could not be replaced")
	}
	afterReportFailure, err := snapshot.HashTree(filepath.Join(project, "content", "docs"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(secondSnapshot, afterReportFailure) {
		t.Fatal("report failure changed the previously valid snapshot")
	}
	if err := os.Remove(reportPath); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(passedPath, []byte("# 现在不合格\n\n![local](images/private.png)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := runConfigured(t.Context(), args, &output, runtime); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "content", "docs", "分类", "安全.md")); !os.IsNotExist(err) {
		t.Fatalf("document that became unsafe survived the next snapshot: %v", err)
	}
}

func TestProductionCLIRejectsSourceAndScannerOverrides(t *testing.T) {
	runtime := runtimeConfig{sourceRoot: "/fixed", gitleaksSHA256: strings.Repeat("a", 64), enforceSource: true}
	for _, argument := range []string{"--source", "--gitleaks"} {
		err := runConfigured(t.Context(), []string{"scan", argument, "/tmp/override"}, &bytes.Buffer{}, runtime)
		if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("override %s was not rejected: %v", argument, err)
		}
	}
}

func TestSourceAndProjectRootsMustNotOverlap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		projectInside bool
	}{
		{name: "project inside source", projectInside: true},
		{name: "source inside project", projectInside: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outer := t.TempDir()
			project := outer
			sourceRoot := filepath.Join(outer, "source")
			if test.projectInside {
				sourceRoot = outer
				project = filepath.Join(outer, "project")
			}
			if err := os.MkdirAll(project, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			_, err := finalizeOptions(commonOptions{projectRoot: project}, true, runtimeConfig{sourceRoot: sourceRoot})
			if err == nil || !strings.Contains(err.Error(), "不得重叠") {
				t.Fatalf("overlapping project/source roots were accepted: %v", err)
			}
		})
	}
}

func hasManifestIndex(manifest snapshot.Manifest, wanted string) bool {
	for _, index := range manifest.GeneratedIndexes {
		if index.Path == wanted {
			return true
		}
	}
	return false
}

func writeFakeGitleaks(t *testing.T, filename string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "--version" ]; then
  echo "8.30.1"
  exit 0
fi
report=""
for argument in "$@"; do
  case "${argument}" in
    --report-path=*) report="${argument#*=}" ;;
  esac
done
if [ -z "${report}" ]; then
  exit 2
fi
printf '[]' > "${report}"
`
	if err := os.WriteFile(filename, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
}

func writeRuleFingerprintFiles(t *testing.T, project string) {
	t.Helper()
	for _, relativePath := range []string{
		"cmd/contentctl/main.go",
		"internal/approval/approval.go",
		"internal/scanner/gitleaks.go",
		"internal/scanner/scanner.go",
		"internal/source/source.go",
		"internal/snapshot/manifest.go",
		"internal/transform/transform.go",
	} {
		filename := filepath.Join(project, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte("synthetic rule source\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func fileHash(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
