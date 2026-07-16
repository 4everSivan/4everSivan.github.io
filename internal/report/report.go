// Package report writes fully redacted, local-only exclusion reports.
package report

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

	"github.com/4everSivan/4everSivan.github.io/internal/approval"
	"github.com/4everSivan/4everSivan.github.io/internal/scanner"
	"gopkg.in/yaml.v3"
)

const SchemaVersion = 1

const LocalPath = ".local/excluded-documents.yaml"

type Status string

const StatusPending Status = "pending"

var ruleIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Report is safe to persist in .local/. It contains document identities and
// redacted finding metadata, never source content or matched values.
type Report struct {
	Version     int        `yaml:"version"`
	GeneratedAt time.Time  `yaml:"generated_at"`
	Documents   []Document `yaml:"documents"`
}

type Document struct {
	Path        string    `yaml:"path"`
	SHA256      string    `yaml:"sha256"`
	Fingerprint string    `yaml:"fingerprint"`
	Findings    []Finding `yaml:"findings"`
	DetectedAt  time.Time `yaml:"detected_at"`
	Status      Status    `yaml:"status"`
}

type Finding struct {
	Fingerprint string        `yaml:"fingerprint"`
	RuleID      string        `yaml:"rule"`
	Level       scanner.Level `yaml:"level"`
	Line        int           `yaml:"line"`
	Reason      string        `yaml:"reason"`
}

// FromResults creates one pending report entry for every document with at
// least one unapproved blocking finding. Scanner-provided reason text is
// deliberately discarded so an accidental secret-bearing message cannot
// enter the report.
func FromResults(results []scanner.Result, generatedAt time.Time, allowlist approval.Allowlist) (Report, error) {
	if generatedAt.IsZero() {
		return Report{}, errors.New("report generation time is required")
	}
	if err := allowlist.Validate(); err != nil {
		return Report{}, fmt.Errorf("validate content scan allowlist: %w", err)
	}
	report := Report{
		Version:     SchemaVersion,
		GeneratedAt: generatedAt.UTC(),
		Documents:   []Document{},
	}
	seenPaths := make(map[string]struct{}, len(results))
	for _, result := range results {
		if !result.Completed {
			return Report{}, fmt.Errorf("scan result for %q is incomplete", result.RelativePath)
		}
		if err := validateRelativePath(result.RelativePath); err != nil {
			return Report{}, err
		}
		if err := validateHash(result.SHA256); err != nil {
			return Report{}, fmt.Errorf("scan result for %q has invalid SHA-256", result.RelativePath)
		}
		if _, duplicate := seenPaths[result.RelativePath]; duplicate {
			return Report{}, fmt.Errorf("duplicate scan result for %q", result.RelativePath)
		}
		seenPaths[result.RelativePath] = struct{}{}

		redactedFindings := make([]Finding, 0)
		for _, finding := range result.Findings {
			if finding.RelativePath != result.RelativePath {
				return Report{}, fmt.Errorf("finding path does not match scan result %q", result.RelativePath)
			}
			if !ruleIDPattern.MatchString(finding.RuleID) || finding.Line < 0 {
				return Report{}, fmt.Errorf("scan result for %q contains invalid finding metadata", result.RelativePath)
			}
			if finding.Level != scanner.LevelBlock && finding.Level != scanner.LevelWarning {
				return Report{}, fmt.Errorf("scan result for %q contains an unknown finding level", result.RelativePath)
			}
			if finding.Level != scanner.LevelBlock || allowlist.Allows(result, finding, generatedAt) {
				continue
			}
			redactedFindings = append(redactedFindings, Finding{
				Fingerprint: approval.FindingFingerprint(finding),
				RuleID:      finding.RuleID,
				Level:       finding.Level,
				Line:        finding.Line,
				Reason:      safeReason(finding.RuleID),
			})
		}
		if len(redactedFindings) == 0 {
			continue
		}
		sortFindings(redactedFindings)
		document := Document{
			Path:       result.RelativePath,
			SHA256:     result.SHA256,
			Findings:   redactedFindings,
			DetectedAt: generatedAt.UTC(),
			Status:     StatusPending,
		}
		document.Fingerprint = documentFingerprint(document)
		report.Documents = append(report.Documents, document)
	}
	sortDocuments(report.Documents)
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

// Validate enforces the redacted schema, exact pending status and duplicate
// protection before any file is replaced.
func (report Report) Validate() error {
	if report.Version != SchemaVersion {
		return fmt.Errorf("unsupported report version %d", report.Version)
	}
	if report.GeneratedAt.IsZero() {
		return errors.New("report generation time is required")
	}
	seenPaths := make(map[string]struct{}, len(report.Documents))
	for documentIndex, document := range report.Documents {
		if err := validateRelativePath(document.Path); err != nil {
			return fmt.Errorf("document %d: %w", documentIndex, err)
		}
		if err := validateHash(document.SHA256); err != nil {
			return fmt.Errorf("document %d has invalid SHA-256", documentIndex)
		}
		if err := validateHash(document.Fingerprint); err != nil {
			return fmt.Errorf("document %d has invalid fingerprint", documentIndex)
		}
		if len(document.Findings) == 0 {
			return fmt.Errorf("document %d has no blocking findings", documentIndex)
		}
		if document.DetectedAt.IsZero() || !document.DetectedAt.Equal(report.GeneratedAt) || document.Status != StatusPending {
			return fmt.Errorf("document %d must have a time and pending status", documentIndex)
		}
		if document.Fingerprint != documentFingerprint(document) {
			return fmt.Errorf("document %d fingerprint does not match redacted findings", documentIndex)
		}
		if _, duplicate := seenPaths[document.Path]; duplicate {
			return fmt.Errorf("document %d duplicates path %q", documentIndex, document.Path)
		}
		seenPaths[document.Path] = struct{}{}

		seenFindings := make(map[string]struct{}, len(document.Findings))
		for findingIndex, finding := range document.Findings {
			if err := validateFinding(finding); err != nil {
				return fmt.Errorf("document %d finding %d: %w", documentIndex, findingIndex, err)
			}
			if _, duplicate := seenFindings[finding.Fingerprint]; duplicate {
				return fmt.Errorf("document %d finding %d duplicates a fingerprint", documentIndex, findingIndex)
			}
			seenFindings[finding.Fingerprint] = struct{}{}
		}
	}
	return nil
}

// Load reads a strict single-document YAML report.
func Load(filePath string) (Report, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Report{}, fmt.Errorf("open exclusion report: %w", err)
	}
	defer file.Close()

	var report Report
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("decode exclusion report: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Report{}, errors.New("multiple YAML documents are not allowed")
		}
		return Report{}, fmt.Errorf("decode trailing report data: %w", err)
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	sortDocuments(report.Documents)
	for index := range report.Documents {
		sortFindings(report.Documents[index].Findings)
		report.Documents[index].Fingerprint = documentFingerprint(report.Documents[index])
	}
	return report, nil
}

// Save atomically replaces a local report with owner-only permissions.
func (report Report) Save(filePath string) error {
	if !isLocalReportPath(filePath) {
		return fmt.Errorf("exclusion report must be saved as %s", LocalPath)
	}
	if err := report.Validate(); err != nil {
		return err
	}
	copyOfDocuments := append([]Document(nil), report.Documents...)
	for index := range copyOfDocuments {
		copyOfDocuments[index].Findings = append([]Finding(nil), copyOfDocuments[index].Findings...)
		sortFindings(copyOfDocuments[index].Findings)
		copyOfDocuments[index].Fingerprint = documentFingerprint(copyOfDocuments[index])
	}
	sortDocuments(copyOfDocuments)
	report.Documents = copyOfDocuments
	return atomicWriteYAML(filePath, report)
}

func validateFinding(finding Finding) error {
	if err := validateHash(finding.Fingerprint); err != nil {
		return errors.New("finding fingerprint must be a lower-case SHA-256 value")
	}
	if !ruleIDPattern.MatchString(finding.RuleID) {
		return errors.New("finding rule identifier is invalid")
	}
	if finding.Level != scanner.LevelBlock {
		return errors.New("exclusion report may only contain blocking findings")
	}
	if finding.Line < 0 {
		return errors.New("finding line cannot be negative")
	}
	if finding.Reason != safeReason(finding.RuleID) {
		return errors.New("finding reason is not the canonical redacted reason")
	}
	return nil
}

func safeReason(ruleID string) string {
	switch ruleID {
	case "secret.private-key":
		return "命中私钥特征"
	case "secret.high-confidence-token":
		return "命中高置信凭据特征"
	case "secret.credential-assignment":
		return "命中疑似凭据配置"
	case "network.private-address":
		return "命中非公开网络地址"
	case "path.absolute-local", "path.file-url", "path.relative-escape":
		return "命中不安全的本地路径引用"
	case "resource.local":
		return "命中未采集的本地资源引用"
	case "syntax.wiki-link":
		return "命中未解析的 Wiki 链接"
	case "html.dangerous":
		return "命中危险 HTML 结构"
	case "content.invalid-utf8", "content.binary":
		return "内容编码或格式不可安全解析"
	case "markdown.invalid-front-matter", "markdown.unclosed-fence":
		return "Markdown 结构不可安全解析"
	default:
		if strings.HasPrefix(ruleID, "gitleaks.") {
			return "命中 Gitleaks 凭据规则"
		}
		return "命中安全或兼容性规则"
	}
}

func documentFingerprint(document Document) string {
	parts := []string{document.Path, document.SHA256}
	for _, finding := range document.Findings {
		parts = append(parts,
			finding.Fingerprint,
			finding.RuleID,
			string(finding.Level),
			fmt.Sprintf("%d", finding.Line),
			finding.Reason,
		)
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].RuleID != findings[j].RuleID {
			return findings[i].RuleID < findings[j].RuleID
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Fingerprint < findings[j].Fingerprint
	})
}

func sortDocuments(documents []Document) {
	sort.SliceStable(documents, func(i, j int) bool {
		return documents[i].Path < documents[j].Path
	})
}

func validateRelativePath(relativePath string) error {
	if relativePath == "" || relativePath == "." || path.IsAbs(relativePath) || strings.Contains(relativePath, "\\") {
		return errors.New("report path must be a slash-separated relative path")
	}
	clean := path.Clean(relativePath)
	if clean != relativePath || clean == ".." || strings.HasPrefix(clean, "../") {
		return errors.New("report path escapes its content root")
	}
	if path.Ext(clean) != ".md" {
		return errors.New("report path must identify a lower-case .md document")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "" || strings.HasPrefix(part, ".") {
			return errors.New("report path contains a hidden or empty component")
		}
	}
	return nil
}

func isLocalReportPath(filePath string) bool {
	clean := filepath.Clean(filePath)
	return filepath.Base(clean) == filepath.Base(LocalPath) && filepath.Base(filepath.Dir(clean)) == filepath.Dir(LocalPath)
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

func atomicWriteYAML(filePath string, value any) error {
	directory := filepath.Dir(filePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("inspect report directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("report directory must not be a symbolic link")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("restrict report directory permissions: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".excluded-documents-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary report: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("set temporary report permissions: %w", err)
	}
	encoder := yaml.NewEncoder(temporary)
	encoder.SetIndent(2)
	if err := encoder.Encode(value); err != nil {
		encoder.Close()
		temporary.Close()
		return fmt.Errorf("encode exclusion report: %w", err)
	}
	if err := encoder.Close(); err != nil {
		temporary.Close()
		return fmt.Errorf("close exclusion report encoder: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary exclusion report: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary exclusion report: %w", err)
	}
	if err := os.Rename(temporaryPath, filePath); err != nil {
		return fmt.Errorf("replace exclusion report: %w", err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open report directory for sync: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync report directory: %w", err)
	}
	return nil
}
