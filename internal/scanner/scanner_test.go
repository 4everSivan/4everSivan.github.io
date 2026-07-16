package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/4everSivan/4everSivan.github.io/internal/source"
)

func TestScanBytesDetectsBlockingRulesWithoutRetainingMatch(t *testing.T) {
	syntheticToken := "gh" + "p_" + strings.Repeat("A", 24)
	body := strings.Join([]string{
		"# Synthetic",
		"pass" + "word = " + strings.Repeat("x", 12),
		"pass" + "word = \"correct horse\"",
		"postgres://demo:" + "long phrase" + "@db.example.invalid/app",
		syntheticToken,
		"-----BEGIN " + "PRIVATE KEY-----",
		"service at " + "10." + "20.30.40",
		"local path /Users/example/private/note.txt",
		"path=/Users/example/private/assigned.txt",
		"config at /srv/private/db.conf",
		"data path=/data/database/cluster",
		`workspace D:\workspace\project\config.yaml`,
		"<img src=/Users/example/private/image.png>",
		"[[ambiguous target]]",
		"<script>synthetic()</script>",
	}, "\n")
	result := ScanBytes("guide/test.md", strings.Repeat("0", 64), []byte(body))

	want := []string{
		RuleCredentialAssignment,
		RuleHighConfidenceToken,
		RulePrivateKey,
		RulePrivateNetwork,
		RuleAbsoluteLocalPath,
		RuleWikiLink,
		RuleDangerousHTML,
	}
	for _, rule := range want {
		if !hasRule(result.Findings, rule) {
			t.Errorf("missing rule %q", rule)
		}
	}
	if !result.HasBlocking() {
		t.Fatal("expected blocking result")
	}
	credential, ok := findRule(result.Findings, RuleCredentialAssignment)
	if !ok || credential.Approvable {
		t.Fatal("credential findings must be blocking and never approvable")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	for _, forbidden := range []string{syntheticToken, strings.Repeat("x", 12), "10." + "20.30.40"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatal("scan result retained a matched value")
		}
	}
}

func TestScanBytesDetectsCommonAbsoluteFilesystemPaths(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"/srv/private/db.conf",
		"path=/data/database/cluster",
		`D:\workspace\project\config.yaml`,
		`C:/workspace/project/config.yaml`,
	} {
		result := ScanBytes("guide/path.md", strings.Repeat("4", 64), []byte(body))
		if !hasRule(result.Findings, RuleAbsoluteLocalPath) {
			t.Fatalf("absolute filesystem path was not detected: %q", body)
		}
	}
}

func TestScanBytesResourceRules(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantRule  string
		wantLevel Level
	}{
		{name: "remote image warning", body: "![image](https://example.com/a.png)", wantRule: RuleRemoteImage, wantLevel: LevelWarning},
		{name: "local image", body: "![image](../images/a.png)", wantRule: RuleLocalResource, wantLevel: LevelBlock},
		{name: "relative escape", body: "[outside](../../../outside.md)", wantRule: RuleRelativePathEscape, wantLevel: LevelBlock},
		{name: "file URL", body: "[local](file:///tmp/note.md)", wantRule: RuleFileURL, wantLevel: LevelBlock},
		{name: "local binary resource", body: "[archive](files/data.zip)", wantRule: RuleLocalResource, wantLevel: LevelBlock},
		{name: "reference definition resource", body: "[asset]: ../files/data.zip", wantRule: RuleLocalResource, wantLevel: LevelBlock},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ScanBytes("guide/note.md", strings.Repeat("1", 64), []byte(test.body))
			finding, ok := findRule(result.Findings, test.wantRule)
			if !ok {
				t.Fatalf("missing rule %q", test.wantRule)
			}
			if finding.Level != test.wantLevel {
				t.Fatalf("level = %q, want %q", finding.Level, test.wantLevel)
			}
		})
	}
}

func TestScanBytesAllowsRemoteAndContainedMarkdownLinks(t *testing.T) {
	body := strings.Join([]string{
		"[sibling](../other.md)",
		"[site](https://example.com/docs)",
		"[anchor](#section)",
		"[email](mailto:reader@example.com)",
	}, "\n")
	result := ScanBytes("guide/note.md", strings.Repeat("2", 64), []byte(body))
	if result.HasBlocking() {
		t.Fatalf("unexpected blocking findings: %#v", result.Blocking())
	}
}

func TestScanBytesRejectsUnparseableContent(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		rule string
	}{
		{name: "invalid utf8", body: []byte{0xff, 0xfe}, rule: RuleInvalidUTF8},
		{name: "binary", body: []byte("first\nsecond\x00"), rule: RuleBinaryContent},
		{name: "front matter", body: []byte("---\ntitle: open\nbody"), rule: RuleInvalidFrontMatter},
		{name: "fence", body: []byte("# title\n```text\nopen"), rule: RuleUnclosedFence},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ScanBytes("note.md", strings.Repeat("3", 64), test.body)
			finding, ok := findRule(result.Findings, test.rule)
			if !ok {
				t.Fatalf("missing rule %q", test.rule)
			}
			if finding.Approvable {
				t.Fatalf("structural rule %q must not be approvable", test.rule)
			}
		})
	}
}

func TestEngineRequiresExternalScanner(t *testing.T) {
	engine := New(nil)
	if err := engine.Check(context.Background()); !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestEngineCompletesPerFileScanAndRevalidatesSource(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# synthetic\n"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	manifest, err := source.Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	external := &fakeExternalScanner{}
	result, err := New(external).Scan(context.Background(), manifest.Root, manifest.Candidates[0])
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if !result.Completed {
		t.Fatal("full scan did not mark result completed")
	}
	if external.scans != 1 {
		t.Fatalf("external scans = %d, want 1", external.scans)
	}
	if string(external.data) != "# synthetic\n" {
		t.Fatal("external scanner did not receive the exact source bytes")
	}
}

type fakeExternalScanner struct {
	scans int
	data  []byte
}

func (f *fakeExternalScanner) Check(context.Context) error { return nil }

func (f *fakeExternalScanner) ScanData(_ context.Context, relativePath string, data []byte) ([]Finding, error) {
	f.scans++
	f.data = append([]byte(nil), data...)
	return []Finding{{
		RuleID:       GitleaksRulePrefix + "synthetic-rule",
		Level:        LevelWarning,
		RelativePath: relativePath,
		Line:         1,
		Reason:       "合成警告",
	}}, nil
}

func hasRule(findings []Finding, rule string) bool {
	_, ok := findRule(findings, rule)
	return ok
}

func findRule(findings []Finding, rule string) (Finding, bool) {
	for _, finding := range findings {
		if finding.RuleID == rule {
			return finding, true
		}
	}
	return Finding{}, false
}
