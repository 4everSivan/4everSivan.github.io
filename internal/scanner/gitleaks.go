package scanner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

const RequiredGitleaksVersion = "8.30.1"

var requiredGitleaksDigests = map[string]string{
	"darwin/arm64": "ba52fb1bfabbcde42f032afad3d6e0b19dff8ed105229a16e7caa338bbc0e84f",
	"linux/amd64":  "88f91962aa2f93ac6ab281d553b9e125f5197bbbce38f9f2437f7299c32e5509",
	"linux/arm64":  "00e91bbe655bd7c47753e8cfe61cb76ea1a5d7e7702fe161ee40102b46b3823b",
}

// RequiredGitleaksSHA256 returns the pinned executable digest for this
// platform. Archive checksums alone are not sufficient because contentctl
// executes the extracted file.
func RequiredGitleaksSHA256() (string, error) {
	digest, ok := requiredGitleaksDigests[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return "", fmt.Errorf("unsupported Gitleaks platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return digest, nil
}

// GitleaksRunner makes a private verified copy of the executable and config,
// then scans caller-provided bytes from a 0700 temporary directory. It never
// asks Gitleaks to reopen a mutable source document.
type GitleaksRunner struct {
	Binary          string
	ExpectedVersion string
	ExpectedSHA256  string
	ConfigPath      string

	mu                sync.Mutex
	preparedDirectory string
	trustedBinary     string
	trustedConfig     string
	configSHA256      string
}

func NewGitleaksRunner(binary, configPath, expectedSHA256 string) *GitleaksRunner {
	return &GitleaksRunner{
		Binary:          binary,
		ExpectedVersion: RequiredGitleaksVersion,
		ExpectedSHA256:  expectedSHA256,
		ConfigPath:      configPath,
	}
}

func (g *GitleaksRunner) Check(ctx context.Context) error {
	if g == nil {
		return ErrScannerUnavailable
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.prepareLocked(ctx); err != nil {
		return err
	}
	return g.verifyInputsLocked()
}

// Close removes the private executable/config copies. It is safe to call more
// than once.
func (g *GitleaksRunner) Close() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.preparedDirectory == "" {
		return nil
	}
	err := os.RemoveAll(g.preparedDirectory)
	g.preparedDirectory = ""
	g.trustedBinary = ""
	g.trustedConfig = ""
	g.configSHA256 = ""
	return err
}

func (g *GitleaksRunner) ScanData(ctx context.Context, relativePath string, data []byte) ([]Finding, error) {
	if g == nil {
		return nil, ErrScannerUnavailable
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.prepareLocked(ctx); err != nil {
		return nil, err
	}
	if err := g.verifyInputsLocked(); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "contentctl-gitleaks-scan-")
	if err != nil {
		return nil, scannerError("create private scanner directory", err)
	}
	defer os.RemoveAll(tempDir)
	if err := os.Chmod(tempDir, 0o700); err != nil {
		return nil, scannerError("protect private scanner directory", err)
	}
	candidatePath := filepath.Join(tempDir, "candidate.md")
	if err := os.WriteFile(candidatePath, data, 0o600); err != nil {
		return nil, scannerError("create private scanner candidate", err)
	}
	reportPath := filepath.Join(tempDir, "report.json")
	templatePath := filepath.Join(tempDir, "safe-report.tmpl")
	const safeReportTemplate = `[{{ range $index, $finding := . }}{{ if $index }},{{ end }}{"RuleID":{{ quote .RuleID }},"StartLine":{{ .StartLine }}}{{ end }}]`
	if err := os.WriteFile(templatePath, []byte(safeReportTemplate), 0o600); err != nil {
		return nil, scannerError("create safe gitleaks report template", err)
	}

	args := []string{
		"dir",
		"--no-banner",
		"--no-color",
		"--log-level=error",
		"--redact=100",
		"--exit-code=0",
		"--report-format=template",
		"--report-path=" + reportPath,
		"--report-template=" + templatePath,
		"--max-archive-depth=0",
		"--max-decode-depth=1",
		"--config=" + g.trustedConfig,
		candidatePath,
	}
	command := exec.CommandContext(ctx, g.trustedBinary, args...)
	command.Stdout = &bytes.Buffer{}
	command.Stderr = &bytes.Buffer{}
	if err := command.Run(); err != nil {
		return nil, scannerError("scan candidate with gitleaks", err)
	}
	if digestBytes(data) != digestFile(candidatePath) {
		return nil, errors.New("private Gitleaks candidate changed during scan")
	}
	if err := g.verifyInputsLocked(); err != nil {
		return nil, err
	}

	reportBytes, err := os.ReadFile(reportPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, scannerError("read redacted gitleaks report", err)
	}
	if len(bytes.TrimSpace(reportBytes)) == 0 {
		return nil, nil
	}
	var report []struct {
		RuleID    string `json:"RuleID"`
		StartLine int    `json:"StartLine"`
	}
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		return nil, scannerError("parse redacted gitleaks report", err)
	}
	findings := make([]Finding, 0, len(report))
	for _, item := range report {
		findings = append(findings, Finding{
			RuleID:       GitleaksRulePrefix + sanitizeRuleID(item.RuleID),
			Level:        LevelBlock,
			RelativePath: relativePath,
			Line:         max(item.StartLine, 0),
			Reason:       "Gitleaks 检测到疑似凭据",
			Approvable:   false,
		})
	}
	return normalizeFindings(findings), nil
}

func (g *GitleaksRunner) prepareLocked(ctx context.Context) error {
	if g.preparedDirectory != "" {
		return nil
	}
	if g.Binary == "" || g.ConfigPath == "" || g.ExpectedVersion == "" || !validDigest(g.ExpectedSHA256) {
		return ErrScannerUnavailable
	}
	binary, binaryDigest, err := readStableRegular(g.Binary)
	if err != nil || binaryDigest != g.ExpectedSHA256 {
		return fmt.Errorf("Gitleaks executable integrity check failed: %w", ErrScannerUnavailable)
	}
	config, configDigest, err := readStableRegular(g.ConfigPath)
	if err != nil {
		return fmt.Errorf("Gitleaks config integrity check failed: %w", ErrScannerUnavailable)
	}
	directory, err := os.MkdirTemp("", "contentctl-gitleaks-runtime-")
	if err != nil {
		return scannerError("create trusted Gitleaks directory", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		os.RemoveAll(directory)
		return scannerError("protect trusted Gitleaks directory", err)
	}
	trustedBinary := filepath.Join(directory, "gitleaks")
	trustedConfig := filepath.Join(directory, "gitleaks.toml")
	if err := os.WriteFile(trustedBinary, binary, 0o500); err != nil {
		os.RemoveAll(directory)
		return scannerError("copy trusted Gitleaks executable", err)
	}
	if err := os.WriteFile(trustedConfig, config, 0o600); err != nil {
		os.RemoveAll(directory)
		return scannerError("copy trusted Gitleaks config", err)
	}
	if digestFile(trustedBinary) != g.ExpectedSHA256 || digestFile(trustedConfig) != configDigest {
		os.RemoveAll(directory)
		return fmt.Errorf("trusted Gitleaks copy integrity check failed: %w", ErrScannerUnavailable)
	}
	command := exec.CommandContext(ctx, trustedBinary, "--version")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &bytes.Buffer{}
	if err := command.Run(); err != nil {
		os.RemoveAll(directory)
		return scannerError("check gitleaks version", err)
	}
	versionPattern := regexp.MustCompile(`(?:^|[^0-9])v?` + regexp.QuoteMeta(g.ExpectedVersion) + `(?:[^0-9]|$)`)
	if !versionPattern.MatchString(output.String()) {
		os.RemoveAll(directory)
		return fmt.Errorf("gitleaks version does not match required %s: %w", g.ExpectedVersion, ErrScannerUnavailable)
	}
	g.preparedDirectory = directory
	g.trustedBinary = trustedBinary
	g.trustedConfig = trustedConfig
	g.configSHA256 = configDigest
	return nil
}

func (g *GitleaksRunner) verifyInputsLocked() error {
	_, binaryDigest, err := readStableRegular(g.Binary)
	if err != nil || binaryDigest != g.ExpectedSHA256 || digestFile(g.trustedBinary) != g.ExpectedSHA256 {
		return fmt.Errorf("Gitleaks executable changed during scan: %w", ErrScannerUnavailable)
	}
	_, configDigest, err := readStableRegular(g.ConfigPath)
	if err != nil || configDigest != g.configSHA256 || digestFile(g.trustedConfig) != g.configSHA256 {
		return fmt.Errorf("Gitleaks config changed during scan: %w", ErrScannerUnavailable)
	}
	return nil
}

func readStableRegular(filename string) ([]byte, string, error) {
	before, err := os.Lstat(filename)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, "", ErrScannerUnavailable
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		file.Close()
		return nil, "", ErrScannerUnavailable
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, "", readErr
	}
	if closeErr != nil {
		return nil, "", closeErr
	}
	after, err := os.Lstat(filename)
	if err != nil || !os.SameFile(before, after) || before.ModTime() != after.ModTime() || before.Size() != after.Size() {
		return nil, "", ErrScannerUnavailable
	}
	return data, digestBytes(data), nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func digestFile(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}
	return digestBytes(data)
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sanitizeRuleID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9', char == '.', char == '-', char == '_':
			builder.WriteRune(char)
		default:
			builder.WriteByte('-')
		}
	}
	clean := strings.Trim(builder.String(), "-")
	if clean == "" {
		return "unknown"
	}
	return clean
}
