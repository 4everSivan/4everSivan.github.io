package scanner

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGitleaksRunnerUsesPinnedRedactedScanAndCleansReport(t *testing.T) {
	dir := t.TempDir()
	tracker := filepath.Join(dir, "report-location")
	argsLog := filepath.Join(dir, "arguments")
	binary := writeFakeGitleaks(t, dir, `
if [ "$1" = "--version" ]; then
  printf '%s\n' 'gitleaks version 8.30.1'
  exit 0
fi
report=''
for arg in "$@"; do
  printf '%s\n' "$arg" >> "$args_log"
  case "$arg" in
    --report-path=*) report=${arg#--report-path=} ;;
  esac
done
printf '%s' "$report" > "$tracker"
printf '%s' '[{"RuleID":"generic-api-key","StartLine":7}]' > "$report"
printf '%s\n' 'SYNTHETIC-SENSITIVE-VALUE' >&2
exit 0
`, tracker, argsLog)
	config := writeTestGitleaksConfig(t, dir)
	runner := NewGitleaksRunner(binary, config, digestFile(binary))
	defer runner.Close()
	findings, err := runner.ScanData(context.Background(), "note.md", []byte("synthetic"))
	if err != nil {
		t.Fatalf("ScanFile() error = %v", err)
	}
	if len(findings) != 1 || findings[0].RuleID != GitleaksRulePrefix+"generic-api-key" || findings[0].Line != 7 {
		t.Fatalf("unexpected redacted findings: %#v", findings)
	}
	encoded := findings[0].Reason + findings[0].RuleID
	if strings.Contains(encoded, "SYNTHETIC-SENSITIVE-VALUE") {
		t.Fatal("finding retained scanner secret")
	}
	reportLocation, err := os.ReadFile(tracker)
	if err != nil {
		t.Fatalf("read report tracker: %v", err)
	}
	if _, err := os.Stat(string(reportLocation)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary report still exists or stat failed: %v", err)
	}
	arguments, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatalf("read arguments: %v", err)
	}
	for _, required := range []string{"--redact=100", "--exit-code=0", "--report-format=template", "--report-template=", "--max-archive-depth=0"} {
		if !strings.Contains(string(arguments), required) {
			t.Errorf("missing required argument %q", required)
		}
	}
}

func TestGitleaksRunnerSuppressesProcessOutputOnFailure(t *testing.T) {
	dir := t.TempDir()
	binary := writeFakeGitleaks(t, dir, `
if [ "$1" = "--version" ]; then
  printf '%s\n' '8.30.1'
  exit 0
fi
printf '%s\n' 'SYNTHETIC-SENSITIVE-VALUE' >&2
exit 7
`, filepath.Join(dir, "tracker"), filepath.Join(dir, "args"))
	runner := NewGitleaksRunner(binary, writeTestGitleaksConfig(t, dir), digestFile(binary))
	defer runner.Close()
	_, err := runner.ScanData(context.Background(), "note.md", []byte("synthetic"))
	if err == nil || !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("ScanFile() error = %v, want scanner unavailable", err)
	}
	if strings.Contains(err.Error(), "SYNTHETIC-SENSITIVE-VALUE") {
		t.Fatal("process error leaked scanner output")
	}
}

func TestGitleaksRunnerRejectsUnexpectedVersionWithoutEchoingOutput(t *testing.T) {
	dir := t.TempDir()
	binary := writeFakeGitleaks(t, dir, `
printf '%s\n' 'unexpected-build-with-private-metadata'
exit 0
`, filepath.Join(dir, "tracker"), filepath.Join(dir, "args"))
	runner := NewGitleaksRunner(binary, writeTestGitleaksConfig(t, dir), digestFile(binary))
	defer runner.Close()
	err := runner.Check(context.Background())
	if err == nil || !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("Check() error = %v", err)
	}
	if strings.Contains(err.Error(), "private-metadata") {
		t.Fatal("version mismatch leaked process output")
	}
}

func TestGitleaksRunnerRejectsSameVersionExecutableWithWrongDigest(t *testing.T) {
	dir := t.TempDir()
	binary := writeFakeGitleaks(t, dir, `
printf '%s\n' '8.30.1'
exit 0
`, filepath.Join(dir, "tracker"), filepath.Join(dir, "args"))
	runner := NewGitleaksRunner(binary, writeTestGitleaksConfig(t, dir), strings.Repeat("a", 64))
	if err := runner.Check(context.Background()); err == nil || !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("Check() error = %v, want integrity failure", err)
	}
}

func TestPinnedExecutableDigestsMatchVersionConfiguration(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "config", "versions.env"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok {
			values[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if values["GITLEAKS_VERSION"] != RequiredGitleaksVersion {
		t.Fatal("Gitleaks version drifted between code and config")
	}
	want := map[string]string{
		"darwin/arm64": values["GITLEAKS_DARWIN_ARM64_BINARY_SHA256"],
		"linux/amd64":  values["GITLEAKS_LINUX_AMD64_BINARY_SHA256"],
		"linux/arm64":  values["GITLEAKS_LINUX_ARM64_BINARY_SHA256"],
	}
	for platform, digest := range requiredGitleaksDigests {
		if want[platform] != digest || !validDigest(digest) {
			t.Fatalf("Gitleaks digest drift for %s", platform)
		}
	}
}

func writeTestGitleaksConfig(t *testing.T, dir string) string {
	t.Helper()
	filename := filepath.Join(dir, ".gitleaks.toml")
	if err := os.WriteFile(filename, []byte("[extend]\nuseDefault = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return filename
}

func writeFakeGitleaks(t *testing.T, dir, body, tracker, argsLog string) string {
	t.Helper()
	path := filepath.Join(dir, "gitleaks-fake")
	script := "#!/bin/sh\ntracker=" + strconv.Quote(tracker) + "\nargs_log=" + strconv.Quote(argsLog) + "\n" + body
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake gitleaks: %v", err)
	}
	return path
}
