package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/4everSivan/4everSivan.github.io/internal/approval"
	"github.com/4everSivan/4everSivan.github.io/internal/scanner"
)

const syntheticSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

var generatedAt = time.Date(2026, 7, 15, 9, 30, 0, 0, time.UTC)

func TestFromResultsIsRedactedSortedAndPending(t *testing.T) {
	syntheticSensitiveValue := "SYNTHETIC_SECRET_MUST_NOT_APPEAR"
	results := []scanner.Result{
		result("z/后记.md",
			finding("z/后记.md", scanner.RulePrivateKey, scanner.LevelBlock, 12, syntheticSensitiveValue, false),
			finding("z/后记.md", scanner.RuleRemoteImage, scanner.LevelWarning, 3, syntheticSensitiveValue, false),
		),
		result("a/note with space.md",
			finding("a/note with space.md", scanner.RuleLocalResource, scanner.LevelBlock, 8, syntheticSensitiveValue, false),
			finding("a/note with space.md", scanner.RulePrivateNetwork, scanner.LevelBlock, 2, syntheticSensitiveValue, true),
		),
	}
	allowlist := approval.New()
	approvedFinding := results[1].Findings[1]
	outputFinding := approvedFinding
	outputFinding.Line += 4
	if _, err := allowlist.Add(approval.Request{
		SourceResult: results[1], SourceFinding: approvedFinding,
		OutputResult: scanner.Result{
			RelativePath: results[1].RelativePath,
			SHA256:       strings.Repeat("d", 64),
			Findings:     []scanner.Finding{outputFinding},
			Completed:    true,
		},
		OutputFinding: outputFinding,
		Reason:        "示例地址用于网络说明",
		ApprovedAt:    generatedAt.Add(-time.Hour),
		ExpiresAt:     generatedAt.Add(time.Hour),
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	report, err := FromResults(results, generatedAt, allowlist)
	if err != nil {
		t.Fatalf("FromResults() error = %v", err)
	}
	if len(report.Documents) != 2 || report.Documents[0].Path != "a/note with space.md" {
		t.Fatalf("documents = %#v", report.Documents)
	}
	if report.Documents[0].Status != StatusPending || len(report.Documents[0].Findings) != 1 {
		t.Fatalf("first document = %#v", report.Documents[0])
	}
	if report.Documents[0].Findings[0].RuleID != scanner.RuleLocalResource {
		t.Fatalf("reported findings = %#v", report.Documents[0].Findings)
	}
	if strings.Contains(report.Documents[0].Findings[0].Reason, syntheticSensitiveValue) {
		t.Fatal("scanner-provided sensitive reason entered the report model")
	}

	filePath := filepath.Join(t.TempDir(), ".local", "excluded-documents.yaml")
	if err := report.Save(filePath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), syntheticSensitiveValue) {
		t.Fatal("serialized report contains the synthetic sensitive value")
	}
	if strings.Contains(string(data), "remote-image") {
		t.Fatal("warning-only finding was written as an exclusion")
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("report mode = %o, want 600", info.Mode().Perm())
	}
}

func TestSaveLoadIsDeterministic(t *testing.T) {
	results := []scanner.Result{
		result("b.md", finding("b.md", scanner.RulePrivateNetwork, scanner.LevelBlock, 4, "ignored", true)),
		result("a.md", finding("a.md", "gitleaks.SyntheticRule", scanner.LevelBlock, 1, "ignored", false)),
	}
	report, err := FromResults(results, generatedAt, approval.New())
	if err != nil {
		t.Fatalf("FromResults() error = %v", err)
	}
	filePath := filepath.Join(t.TempDir(), ".local", "excluded-documents.yaml")
	if err := report.Save(filePath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	before, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(filePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Documents) != 2 || loaded.Documents[0].Path != "a.md" {
		t.Fatalf("loaded documents = %#v", loaded.Documents)
	}
	if err := loaded.Save(filePath); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	after, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("repeated report save was not deterministic")
	}
}

func TestDocumentFingerprintBindsDisplayedFindingMetadata(t *testing.T) {
	report, err := FromResults([]scanner.Result{
		result("a.md", finding("a.md", scanner.RulePrivateKey, scanner.LevelBlock, 1, "ignored", false)),
	}, generatedAt, approval.New())
	if err != nil {
		t.Fatal(err)
	}
	report.Documents[0].Findings[0].Line++
	if err := report.Validate(); err == nil {
		t.Fatal("Validate() accepted finding metadata changed after fingerprinting")
	}
}

func TestRejectsIncompleteDuplicateAndUnknownData(t *testing.T) {
	incomplete := result("a.md", finding("a.md", scanner.RulePrivateKey, scanner.LevelBlock, 1, "ignored", false))
	incomplete.Completed = false
	if _, err := FromResults([]scanner.Result{incomplete}, generatedAt, approval.New()); err == nil {
		t.Fatal("FromResults() accepted an incomplete scan")
	}

	complete := result("a.md", finding("a.md", scanner.RulePrivateKey, scanner.LevelBlock, 1, "ignored", false))
	if _, err := FromResults([]scanner.Result{complete, complete}, generatedAt, approval.New()); err == nil {
		t.Fatal("FromResults() accepted duplicate document results")
	}
	unknownLevel := result("level.md", finding("level.md", scanner.RulePrivateKey, scanner.Level("future"), 1, "ignored", false))
	if _, err := FromResults([]scanner.Result{unknownLevel}, generatedAt, approval.New()); err == nil {
		t.Fatal("FromResults() silently skipped an unknown finding level")
	}
	invalidAllowlist := approval.New()
	invalidAllowlist.Version++
	if _, err := FromResults([]scanner.Result{complete}, generatedAt, invalidAllowlist); err == nil {
		t.Fatal("FromResults() accepted an invalid allowlist")
	}

	filePath := filepath.Join(t.TempDir(), "report.yaml")
	invalid := "version: 1\ngenerated_at: 2026-07-15T09:30:00Z\ndocuments: []\nunknown: true\n"
	if err := os.WriteFile(filePath, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filePath); err == nil {
		t.Fatal("Load() accepted an unknown field")
	}
}

func TestInvalidSavePreservesOldReport(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), ".local", "excluded-documents.yaml")
	if err := os.Mkdir(filepath.Dir(filePath), 0o700); err != nil {
		t.Fatal(err)
	}
	old := []byte("existing-safe-report\n")
	if err := os.WriteFile(filePath, old, 0o600); err != nil {
		t.Fatal(err)
	}
	invalid := Report{Version: SchemaVersion, GeneratedAt: generatedAt, Documents: []Document{{Path: "../escape.md"}}}
	if err := invalid.Save(filePath); err == nil {
		t.Fatal("Save() accepted an invalid report")
	}
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(old) {
		t.Fatal("invalid report replaced the previous report")
	}
}

func TestReportDirectoryMustNotBeSymlink(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, ".local")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	valid, err := FromResults([]scanner.Result{
		result("a.md", finding("a.md", scanner.RulePrivateKey, scanner.LevelBlock, 1, "ignored", false)),
	}, generatedAt, approval.New())
	if err != nil {
		t.Fatal(err)
	}
	if err := valid.Save(filepath.Join(link, "excluded-documents.yaml")); err == nil {
		t.Fatal("Save() followed a symlinked report directory")
	}
}

func result(relativePath string, findings ...scanner.Finding) scanner.Result {
	return scanner.Result{
		RelativePath: relativePath,
		SHA256:       syntheticSHA256,
		Findings:     findings,
		Completed:    true,
	}
}

func finding(relativePath, rule string, level scanner.Level, line int, reason string, approvable bool) scanner.Finding {
	return scanner.Finding{
		RelativePath: relativePath,
		RuleID:       rule,
		Level:        level,
		Line:         line,
		Reason:       reason,
		Approvable:   approvable,
	}
}
