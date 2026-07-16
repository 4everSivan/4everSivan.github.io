package approval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/4everSivan/4everSivan.github.io/internal/scanner"
)

const syntheticDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const syntheticOutputDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

var syntheticTime = time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)

func TestExactDualIdentityApprovalAndExpiry(t *testing.T) {
	finding := syntheticFinding(scanner.RulePrivateNetwork, true, 7)
	result := syntheticResult(finding)
	request := pairedRequest(result, finding, "示例地址用于解释保留网段")
	allowlist := New()
	entry, err := allowlist.Add(request)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if entry.Path != result.RelativePath || entry.SourceSHA256 != result.SHA256 ||
		entry.OutputSHA256 != request.OutputResult.SHA256 || entry.RuleID != finding.RuleID {
		t.Fatalf("approval identity = %#v", entry)
	}
	at := syntheticTime.Add(time.Hour)
	if !allowlist.Allows(result, finding, at) {
		t.Fatal("exact source approval did not match")
	}
	if !allowlist.AllowsOutput(request.OutputResult, request.OutputFinding, at) {
		t.Fatal("exact output approval did not match")
	}

	changedSource := result
	changedSource.SHA256 = strings.Repeat("c", 64)
	if allowlist.Allows(changedSource, finding, at) {
		t.Fatal("source approval survived a source hash change")
	}
	changedOutput := request.OutputResult
	changedOutput.SHA256 = strings.Repeat("d", 64)
	if allowlist.AllowsOutput(changedOutput, request.OutputFinding, at) {
		t.Fatal("output approval survived an output hash change")
	}
	changedOutputFinding := request.OutputFinding
	changedOutputFinding.Line++
	if allowlist.AllowsOutput(request.OutputResult, changedOutputFinding, at) {
		t.Fatal("output approval survived a finding change")
	}
	if allowlist.Allows(result, finding, syntheticTime.Add(25*time.Hour)) {
		t.Fatal("expired approval remained active")
	}
}

func TestApprovalDoesNotHideOtherBlockingRule(t *testing.T) {
	approvedFinding := syntheticFinding(scanner.RulePrivateNetwork, true, 2)
	structuralFinding := syntheticFinding(scanner.RuleLocalResource, false, 3)
	result := syntheticResult(approvedFinding, structuralFinding)
	allowlist := New()
	if _, err := allowlist.Add(pairedRequest(result, approvedFinding, "示例地址用于解释保留网段")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	remaining := allowlist.UnapprovedBlocking(result, syntheticTime.Add(time.Hour))
	if len(remaining) != 1 || remaining[0].RuleID != scanner.RuleLocalResource {
		t.Fatalf("remaining findings = %#v", remaining)
	}
}

func TestRejectsSecretsStructuralAndUnknownRules(t *testing.T) {
	tests := []struct {
		name   string
		rule   string
		marked bool
	}{
		{name: "credential assignment", rule: scanner.RuleCredentialAssignment, marked: true},
		{name: "private key", rule: scanner.RulePrivateKey, marked: false},
		{name: "high confidence token", rule: scanner.RuleHighConfidenceToken, marked: false},
		{name: "path escape", rule: scanner.RuleRelativePathEscape, marked: false},
		{name: "local resource", rule: scanner.RuleLocalResource, marked: false},
		{name: "invalid content", rule: scanner.RuleInvalidUTF8, marked: false},
		{name: "unknown opt-in", rule: "future.low-risk", marked: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			finding := syntheticFinding(test.rule, test.marked, 1)
			result := syntheticResult(finding)
			allowlist := New()
			if _, err := allowlist.Add(pairedRequest(result, finding, "已完成人工复核并确认是示例")); err == nil {
				t.Fatal("Add() accepted a non-approvable rule")
			}
		})
	}
}

func TestRejectsSensitiveReviewReasonWithoutEchoingIt(t *testing.T) {
	finding := syntheticFinding(scanner.RulePrivateNetwork, true, 1)
	result := syntheticResult(finding)
	syntheticValue := "SYNTHETIC_ONLY_1234567890_VALUE"
	request := pairedRequest(result, finding, "示例地址")
	request.Reason = "token=" + syntheticValue
	allowlist := New()
	_, err := allowlist.Add(request)
	if err == nil {
		t.Fatal("Add() accepted a secret-bearing review reason")
	}
	if strings.Contains(err.Error(), syntheticValue) {
		t.Fatal("validation error echoed the synthetic sensitive value")
	}
}

func TestSaveLoadStrictAndDeterministic(t *testing.T) {
	first := syntheticFinding(scanner.RulePrivateNetwork, true, 9)
	second := syntheticFinding(scanner.RulePrivateNetwork, true, 3)
	first.RelativePath = "z/网络.md"
	second.RelativePath = "a/note with space.md"
	firstResult := syntheticResult(first)
	secondResult := syntheticResult(second)

	allowlist := New()
	for _, request := range []Request{
		pairedRequest(firstResult, first, "示例地址用于协议说明"),
		pairedRequest(secondResult, second, "示例地址用于网络说明"),
	} {
		if _, err := allowlist.Add(request); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	filePath := filepath.Join(t.TempDir(), "config", "content-scan-allowlist.yaml")
	if err := allowlist.Save(filePath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	firstBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(filePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Approvals) != 2 || loaded.Approvals[0].Path != second.RelativePath {
		t.Fatalf("loaded approvals are not sorted: %#v", loaded.Approvals)
	}
	if err := loaded.Save(filePath); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	secondBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("repeated save was not deterministic")
	}

	unknownFieldPath := filepath.Join(t.TempDir(), "unknown.yaml")
	unknown := "version: 2\napprovals: []\nunknown_field: true\n"
	if err := os.WriteFile(unknownFieldPath, []byte(unknown), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(unknownFieldPath); err == nil {
		t.Fatal("Load() accepted an unknown field")
	}
}

func TestVersionedEmptyAllowlistLoads(t *testing.T) {
	allowlist, err := Load(filepath.Join("..", "..", ConfigPath))
	if err != nil {
		t.Fatalf("Load(versioned config) error = %v", err)
	}
	if allowlist.Version != SchemaVersion || len(allowlist.Approvals) != 0 {
		t.Fatalf("versioned allowlist = %#v", allowlist)
	}
}

func TestInvalidSavePreservesExistingFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "allowlist.yaml")
	old := []byte("existing-safe-content\n")
	if err := os.WriteFile(filePath, old, 0o600); err != nil {
		t.Fatal(err)
	}
	invalid := New()
	invalid.Approvals = []Entry{{Path: "../escape.md"}}
	if err := invalid.Save(filePath); err == nil {
		t.Fatal("Save() accepted invalid configuration")
	}
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(old) {
		t.Fatal("invalid save modified the previous file")
	}
}

func pairedRequest(result scanner.Result, finding scanner.Finding, reason string) Request {
	outputFinding := finding
	outputFinding.Line += 4
	outputResult := scanner.Result{
		RelativePath: result.RelativePath,
		SHA256:       syntheticOutputDigest,
		Findings:     []scanner.Finding{outputFinding},
		Completed:    true,
	}
	return Request{
		SourceResult: result, SourceFinding: finding,
		OutputResult: outputResult, OutputFinding: outputFinding,
		Reason: reason, ApprovedAt: syntheticTime, ExpiresAt: syntheticTime.Add(24 * time.Hour),
	}
}

func syntheticFinding(rule string, approvable bool, line int) scanner.Finding {
	return scanner.Finding{
		RuleID:       rule,
		Level:        scanner.LevelBlock,
		RelativePath: "notes/example.md",
		Line:         line,
		Reason:       "固定的脱敏规则说明",
		Approvable:   approvable,
	}
}

func syntheticResult(findings ...scanner.Finding) scanner.Result {
	return scanner.Result{
		RelativePath: findings[0].RelativePath,
		SHA256:       syntheticDigest,
		Findings:     findings,
		Completed:    true,
	}
}
