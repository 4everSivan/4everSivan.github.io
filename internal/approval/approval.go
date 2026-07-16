// Package approval manages explicit, hash-bound approvals for reviewed
// low-risk scanner findings.
package approval

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/4everSivan/4everSivan.github.io/internal/scanner"
	"gopkg.in/yaml.v3"
)

const SchemaVersion = 2

const ConfigPath = "config/content-scan-allowlist.yaml"

var (
	ruleIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

	// The policy is intentionally closed. A scanner boolean alone cannot make
	// a newly added or structural rule approvable.
	lowRiskRules = map[string]struct{}{
		"network.private-address": {},
	}

	unsafeReasonPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`),
		regexp.MustCompile(`(?i)\b(?:password|passwd|token|secret|api[ _-]?key)\b\s*[:=]\s*\S+`),
		regexp.MustCompile(`\b(?:AKIA[0-9A-Z]{16}|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9_-]{20,})\b`),
		regexp.MustCompile(`\b[0-9A-Fa-f]{32,}\b`),
		regexp.MustCompile(`\b[A-Za-z0-9+/]{40,}={0,2}\b`),
	}
)

// Allowlist is the versioned, repository-tracked approval configuration.
type Allowlist struct {
	Version   int     `yaml:"version"`
	Approvals []Entry `yaml:"approvals"`
}

// Entry binds an approval to one exact document and one exact redacted
// finding. It deliberately contains no matched text or sensitive value.
type Entry struct {
	Path                     string        `yaml:"path"`
	SourceSHA256             string        `yaml:"source_sha256"`
	SourceFindingFingerprint string        `yaml:"source_finding_fingerprint"`
	SourceLine               int           `yaml:"source_line"`
	OutputSHA256             string        `yaml:"output_sha256"`
	OutputFindingFingerprint string        `yaml:"output_finding_fingerprint"`
	OutputLine               int           `yaml:"output_line"`
	RuleID                   string        `yaml:"rule"`
	Level                    scanner.Level `yaml:"level"`
	Reason                   string        `yaml:"reason"`
	ApprovedAt               time.Time     `yaml:"approved_at"`
	ExpiresAt                time.Time     `yaml:"expires_at"`
}

// Request is created only after a user explicitly reviews a finding.
type Request struct {
	SourceResult  scanner.Result
	SourceFinding scanner.Finding
	OutputResult  scanner.Result
	OutputFinding scanner.Finding
	Reason        string
	ApprovedAt    time.Time
	ExpiresAt     time.Time
}

// New returns an empty allowlist with the current schema version.
func New() Allowlist {
	return Allowlist{Version: SchemaVersion, Approvals: []Entry{}}
}

// FindingFingerprint creates a stable, non-reversible identity from redacted
// finding metadata. The document path and content hash remain separate exact
// match keys in an Entry.
func FindingFingerprint(finding scanner.Finding) string {
	canonical := fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%s\x00%t",
		finding.RelativePath,
		finding.RuleID,
		finding.Level,
		finding.Line,
		finding.Reason,
		finding.Approvable,
	)
	digest := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(digest[:])
}

// Add validates and adds or replaces an exact approval. It never permits a
// structural or high-confidence secret finding, even if a caller constructs
// an Entry manually.
func (allowlist *Allowlist) Add(request Request) (Entry, error) {
	if allowlist == nil {
		return Entry{}, errors.New("allowlist is nil")
	}
	if allowlist.Version == 0 {
		allowlist.Version = SchemaVersion
	}
	if allowlist.Version != SchemaVersion {
		return Entry{}, fmt.Errorf("unsupported allowlist version %d", allowlist.Version)
	}
	if err := validateRequest(request); err != nil {
		return Entry{}, err
	}

	entry := Entry{
		Path:                     request.SourceResult.RelativePath,
		SourceSHA256:             request.SourceResult.SHA256,
		SourceFindingFingerprint: FindingFingerprint(request.SourceFinding),
		SourceLine:               request.SourceFinding.Line,
		OutputSHA256:             request.OutputResult.SHA256,
		OutputFindingFingerprint: FindingFingerprint(request.OutputFinding),
		OutputLine:               request.OutputFinding.Line,
		RuleID:                   request.SourceFinding.RuleID,
		Level:                    request.SourceFinding.Level,
		Reason:                   strings.TrimSpace(request.Reason),
		ApprovedAt:               request.ApprovedAt.UTC(),
		ExpiresAt:                request.ExpiresAt.UTC(),
	}
	if err := validateEntry(entry); err != nil {
		return Entry{}, err
	}

	key := entryKey(entry)
	replaced := false
	for index := range allowlist.Approvals {
		if entryKey(allowlist.Approvals[index]) == key {
			allowlist.Approvals[index] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		allowlist.Approvals = append(allowlist.Approvals, entry)
	}
	sortEntries(allowlist.Approvals)
	return entry, nil
}

// Allows reports whether a source-side finding has an exact, active approval.
// Output verification must use AllowsOutput; the two identities cannot be
// substituted for one another.
func (allowlist Allowlist) Allows(result scanner.Result, finding scanner.Finding, at time.Time) bool {
	return allowlist.allows(result, finding, at, false)
}

// AllowsOutput verifies an approval against the exact transformed bytes and
// the finding produced by a real scan of those bytes.
func (allowlist Allowlist) AllowsOutput(result scanner.Result, finding scanner.Finding, at time.Time) bool {
	return allowlist.allows(result, finding, at, true)
}

func (allowlist Allowlist) allows(result scanner.Result, finding scanner.Finding, at time.Time, output bool) bool {
	if allowlist.Version != SchemaVersion {
		return false
	}
	if err := validateResultFinding(result, finding); err != nil ||
		!containsFinding(result.Findings, finding) ||
		!canApprove(finding) ||
		at.IsZero() {
		return false
	}
	fingerprint := FindingFingerprint(finding)
	for _, entry := range allowlist.Approvals {
		if entry.Path != result.RelativePath || entry.RuleID != finding.RuleID || entry.Level != finding.Level {
			continue
		}
		if output {
			if entry.OutputSHA256 != result.SHA256 || entry.OutputFindingFingerprint != fingerprint || entry.OutputLine != finding.Line {
				continue
			}
		} else if entry.SourceSHA256 != result.SHA256 || entry.SourceFindingFingerprint != fingerprint || entry.SourceLine != finding.Line {
			continue
		}
		if validateEntry(entry) != nil {
			continue
		}
		instant := at.UTC()
		return !instant.Before(entry.ApprovedAt) && instant.Before(entry.ExpiresAt)
	}
	return false
}

// UnapprovedBlocking returns every blocking finding that still excludes the
// document after exact approvals are applied.
func (allowlist Allowlist) UnapprovedBlocking(result scanner.Result, at time.Time) []scanner.Finding {
	return allowlist.unapprovedBlocking(result, at, false)
}

// UnapprovedOutputBlocking applies output-side bindings only.
func (allowlist Allowlist) UnapprovedOutputBlocking(result scanner.Result, at time.Time) []scanner.Finding {
	return allowlist.unapprovedBlocking(result, at, true)
}

func (allowlist Allowlist) unapprovedBlocking(result scanner.Result, at time.Time, output bool) []scanner.Finding {
	findings := make([]scanner.Finding, 0)
	for _, finding := range result.Findings {
		allowed := allowlist.Allows(result, finding, at)
		if output {
			allowed = allowlist.AllowsOutput(result, finding, at)
		}
		if finding.Level == scanner.LevelBlock && !allowed {
			findings = append(findings, finding)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].RuleID != findings[j].RuleID {
			return findings[i].RuleID < findings[j].RuleID
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return FindingFingerprint(findings[i]) < FindingFingerprint(findings[j])
	})
	return findings
}

// Validate performs strict schema and duplicate checks. Expired approvals are
// valid historical records but Allows will not activate them.
func (allowlist Allowlist) Validate() error {
	if allowlist.Version != SchemaVersion {
		return fmt.Errorf("unsupported allowlist version %d", allowlist.Version)
	}
	seen := make(map[string]struct{}, len(allowlist.Approvals))
	for index, entry := range allowlist.Approvals {
		if err := validateEntry(entry); err != nil {
			return fmt.Errorf("approval %d: %w", index, err)
		}
		key := entryKey(entry)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("approval %d duplicates an exact finding", index)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// Load reads a strict YAML allowlist. Unknown fields and duplicate exact
// entries are rejected rather than ignored.
func Load(filePath string) (Allowlist, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Allowlist{}, fmt.Errorf("open allowlist: %w", err)
	}
	defer file.Close()

	var allowlist Allowlist
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&allowlist); err != nil {
		return Allowlist{}, fmt.Errorf("decode allowlist: %w", err)
	}
	if err := ensureSingleYAMLDocument(decoder); err != nil {
		return Allowlist{}, err
	}
	if err := allowlist.Validate(); err != nil {
		return Allowlist{}, err
	}
	sortEntries(allowlist.Approvals)
	return allowlist, nil
}

// Save validates and atomically replaces a repository-readable YAML file.
func (allowlist Allowlist) Save(filePath string) error {
	if !isConfigPath(filePath) {
		return fmt.Errorf("allowlist must be saved as %s", ConfigPath)
	}
	if err := allowlist.Validate(); err != nil {
		return err
	}
	copyOfEntries := append([]Entry(nil), allowlist.Approvals...)
	sortEntries(copyOfEntries)
	allowlist.Approvals = copyOfEntries
	return atomicWriteYAML(filePath, allowlist, 0o644)
}

func validateRequest(request Request) error {
	if err := validateResultFinding(request.SourceResult, request.SourceFinding); err != nil {
		return err
	}
	if err := validateResultFinding(request.OutputResult, request.OutputFinding); err != nil {
		return err
	}
	if !canApprove(request.SourceFinding) || !canApprove(request.OutputFinding) {
		return fmt.Errorf("rule %q is not eligible for approval", request.SourceFinding.RuleID)
	}
	if !containsFinding(request.SourceResult.Findings, request.SourceFinding) || !containsFinding(request.OutputResult.Findings, request.OutputFinding) {
		return errors.New("finding is not present in both completed scan results")
	}
	if request.SourceResult.RelativePath != request.OutputResult.RelativePath ||
		request.SourceFinding.RuleID != request.OutputFinding.RuleID ||
		request.SourceFinding.Level != request.OutputFinding.Level ||
		request.SourceFinding.Reason != request.OutputFinding.Reason ||
		request.SourceFinding.Approvable != request.OutputFinding.Approvable {
		return errors.New("source and output findings do not describe the same reviewed rule")
	}
	if request.ApprovedAt.IsZero() {
		return errors.New("approval time is required")
	}
	if request.ExpiresAt.IsZero() || !request.ExpiresAt.After(request.ApprovedAt) {
		return errors.New("approval expiry must be after approval time")
	}
	if err := validateReviewReason(request.Reason); err != nil {
		return err
	}
	return nil
}

func validateResultFinding(result scanner.Result, finding scanner.Finding) error {
	if !result.Completed {
		return errors.New("approval requires a completed scan result")
	}
	if err := validateRelativePath(result.RelativePath); err != nil {
		return err
	}
	if err := validateHash(result.SHA256); err != nil {
		return err
	}
	if finding.RelativePath != result.RelativePath {
		return errors.New("finding path does not match scan result")
	}
	if !ruleIDPattern.MatchString(finding.RuleID) {
		return errors.New("finding rule identifier is invalid")
	}
	if finding.Line <= 0 {
		return errors.New("approvable finding line must be positive")
	}
	return nil
}

func canApprove(finding scanner.Finding) bool {
	if finding.Level != scanner.LevelBlock || !finding.Approvable {
		return false
	}
	_, allowed := lowRiskRules[finding.RuleID]
	return allowed
}

func containsFinding(findings []scanner.Finding, wanted scanner.Finding) bool {
	wantedFingerprint := FindingFingerprint(wanted)
	for _, finding := range findings {
		if FindingFingerprint(finding) == wantedFingerprint {
			return true
		}
	}
	return false
}

func validateEntry(entry Entry) error {
	if err := validateRelativePath(entry.Path); err != nil {
		return err
	}
	if err := validateHash(entry.SourceSHA256); err != nil {
		return err
	}
	if err := validateHash(entry.OutputSHA256); err != nil {
		return err
	}
	if !ruleIDPattern.MatchString(entry.RuleID) {
		return errors.New("approval rule identifier is invalid")
	}
	if _, allowed := lowRiskRules[entry.RuleID]; !allowed {
		return fmt.Errorf("rule %q is not eligible for approval", entry.RuleID)
	}
	if err := validateHash(entry.SourceFindingFingerprint); err != nil {
		return errors.New("source finding fingerprint must be a lower-case SHA-256 value")
	}
	if err := validateHash(entry.OutputFindingFingerprint); err != nil {
		return errors.New("output finding fingerprint must be a lower-case SHA-256 value")
	}
	if entry.Level != scanner.LevelBlock {
		return errors.New("only blocking low-risk findings may be approved")
	}
	if entry.SourceLine <= 0 || entry.OutputLine <= 0 {
		return errors.New("approval source/output lines must be positive")
	}
	if err := validateReviewReason(entry.Reason); err != nil {
		return err
	}
	if entry.ApprovedAt.IsZero() {
		return errors.New("approval time is required")
	}
	if entry.ExpiresAt.IsZero() || !entry.ExpiresAt.After(entry.ApprovedAt) {
		return errors.New("approval expiry must be after approval time")
	}
	return nil
}

func validateReviewReason(reason string) error {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" || trimmed != reason {
		return errors.New("approval reason must be non-empty and trimmed")
	}
	if len([]rune(trimmed)) > 240 {
		return errors.New("approval reason is too long")
	}
	for _, character := range trimmed {
		if unicode.IsControl(character) {
			return errors.New("approval reason cannot contain control characters")
		}
	}
	for _, pattern := range unsafeReasonPatterns {
		if pattern.MatchString(trimmed) {
			return errors.New("approval reason appears to contain sensitive material")
		}
	}
	return nil
}

func validateRelativePath(relativePath string) error {
	if relativePath == "" || relativePath == "." || path.IsAbs(relativePath) || strings.Contains(relativePath, "\\") {
		return errors.New("approval path must be a slash-separated relative path")
	}
	clean := path.Clean(relativePath)
	if clean != relativePath || clean == ".." || strings.HasPrefix(clean, "../") {
		return errors.New("approval path escapes its content root")
	}
	if path.Ext(clean) != ".md" {
		return errors.New("approval path must identify a lower-case .md document")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "" || strings.HasPrefix(part, ".") {
			return errors.New("approval path contains a hidden or empty component")
		}
	}
	return nil
}

func isConfigPath(filePath string) bool {
	clean := filepath.Clean(filePath)
	return filepath.Base(clean) == filepath.Base(ConfigPath) && filepath.Base(filepath.Dir(clean)) == filepath.Dir(ConfigPath)
}

func validateHash(value string) error {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return errors.New("value must be a lower-case SHA-256 digest")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return errors.New("value must be a lower-case SHA-256 digest")
	}
	return nil
}

func entryKey(entry Entry) string {
	return strings.Join([]string{
		entry.Path,
		entry.SourceSHA256,
		entry.SourceFindingFingerprint,
		entry.OutputSHA256,
		entry.OutputFindingFingerprint,
		entry.RuleID,
	}, "\x00")
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.SourceSHA256 != right.SourceSHA256 {
			return left.SourceSHA256 < right.SourceSHA256
		}
		if left.RuleID != right.RuleID {
			return left.RuleID < right.RuleID
		}
		if left.SourceFindingFingerprint != right.SourceFindingFingerprint {
			return left.SourceFindingFingerprint < right.SourceFindingFingerprint
		}
		return left.OutputFindingFingerprint < right.OutputFindingFingerprint
	})
}

func ensureSingleYAMLDocument(decoder *yaml.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("multiple YAML documents are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode trailing allowlist data: %w", err)
	}
	return nil
}

func atomicWriteYAML(filePath string, value any, mode os.FileMode) error {
	directory := filepath.Dir(filePath)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create allowlist directory: %w", err)
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("inspect allowlist directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("allowlist directory must not be a symbolic link")
	}

	temporary, err := os.CreateTemp(directory, ".allowlist-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary allowlist: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return fmt.Errorf("set temporary allowlist permissions: %w", err)
	}
	encoder := yaml.NewEncoder(temporary)
	encoder.SetIndent(2)
	if err := encoder.Encode(value); err != nil {
		encoder.Close()
		temporary.Close()
		return fmt.Errorf("encode allowlist: %w", err)
	}
	if err := encoder.Close(); err != nil {
		temporary.Close()
		return fmt.Errorf("close allowlist encoder: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary allowlist: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary allowlist: %w", err)
	}
	if err := os.Rename(temporaryPath, filePath); err != nil {
		return fmt.Errorf("replace allowlist: %w", err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open allowlist directory for sync: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync allowlist directory: %w", err)
	}
	return nil
}
