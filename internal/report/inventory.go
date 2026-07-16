package report

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/4everSivan/4everSivan.github.io/internal/approval"
	"github.com/4everSivan/4everSivan.github.io/internal/scanner"
)

const InventoryPath = ".local/scan-results.yaml"

type InventoryStatus string

const (
	InventoryPassed   InventoryStatus = "passed"
	InventoryExcluded InventoryStatus = "excluded"
)

// Inventory records the complete, redacted classification for one scan run.
type Inventory struct {
	Version        int                 `yaml:"version"`
	GeneratedAt    time.Time           `yaml:"generated_at"`
	CandidateCount int                 `yaml:"candidate_count"`
	PassedCount    int                 `yaml:"passed_count"`
	ExcludedCount  int                 `yaml:"excluded_count"`
	Documents      []InventoryDocument `yaml:"documents"`
}

type InventoryDocument struct {
	Path     string             `yaml:"path"`
	SHA256   string             `yaml:"sha256"`
	Status   InventoryStatus    `yaml:"status"`
	Findings []InventoryFinding `yaml:"findings,omitempty"`
}

type InventoryFinding struct {
	Fingerprint string        `yaml:"fingerprint"`
	RuleID      string        `yaml:"rule"`
	Level       scanner.Level `yaml:"level"`
	Line        int           `yaml:"line"`
	Approved    bool          `yaml:"approved"`
}

func InventoryFromResults(results []scanner.Result, generatedAt time.Time, allowlist approval.Allowlist) (Inventory, error) {
	if generatedAt.IsZero() {
		return Inventory{}, errors.New("inventory generation time is required")
	}
	if err := allowlist.Validate(); err != nil {
		return Inventory{}, fmt.Errorf("validate content scan allowlist: %w", err)
	}
	inventory := Inventory{Version: SchemaVersion, GeneratedAt: generatedAt.UTC(), CandidateCount: len(results)}
	seen := make(map[string]struct{}, len(results))
	for _, result := range results {
		if !result.Completed {
			return Inventory{}, fmt.Errorf("scan result for %q is incomplete", result.RelativePath)
		}
		if err := validateRelativePath(result.RelativePath); err != nil {
			return Inventory{}, err
		}
		if err := validateHash(result.SHA256); err != nil {
			return Inventory{}, fmt.Errorf("scan result for %q has invalid SHA-256", result.RelativePath)
		}
		if _, duplicate := seen[result.RelativePath]; duplicate {
			return Inventory{}, fmt.Errorf("duplicate scan result for %q", result.RelativePath)
		}
		seen[result.RelativePath] = struct{}{}

		document := InventoryDocument{Path: result.RelativePath, SHA256: result.SHA256, Status: InventoryPassed}
		for _, finding := range result.Findings {
			if finding.RelativePath != result.RelativePath || finding.Line < 0 {
				return Inventory{}, fmt.Errorf("invalid finding metadata for %q", result.RelativePath)
			}
			approved := allowlist.Allows(result, finding, generatedAt)
			if finding.Level == scanner.LevelBlock && !approved {
				document.Status = InventoryExcluded
			}
			document.Findings = append(document.Findings, InventoryFinding{
				Fingerprint: approval.FindingFingerprint(finding),
				RuleID:      finding.RuleID, Level: finding.Level, Line: finding.Line, Approved: approved,
			})
		}
		sort.Slice(document.Findings, func(i, j int) bool {
			if document.Findings[i].RuleID != document.Findings[j].RuleID {
				return document.Findings[i].RuleID < document.Findings[j].RuleID
			}
			if document.Findings[i].Line != document.Findings[j].Line {
				return document.Findings[i].Line < document.Findings[j].Line
			}
			return document.Findings[i].Fingerprint < document.Findings[j].Fingerprint
		})
		if document.Status == InventoryExcluded {
			inventory.ExcludedCount++
		} else {
			inventory.PassedCount++
		}
		inventory.Documents = append(inventory.Documents, document)
	}
	sort.Slice(inventory.Documents, func(i, j int) bool { return inventory.Documents[i].Path < inventory.Documents[j].Path })
	if inventory.PassedCount+inventory.ExcludedCount != inventory.CandidateCount {
		return Inventory{}, errors.New("inventory classification is incomplete")
	}
	return inventory, nil
}

func (inventory Inventory) Save(filePath string) error {
	clean := filepath.Clean(filePath)
	if filepath.Base(clean) != filepath.Base(InventoryPath) || filepath.Base(filepath.Dir(clean)) != filepath.Dir(InventoryPath) {
		return fmt.Errorf("scan inventory must be saved as %s", InventoryPath)
	}
	if inventory.Version != SchemaVersion || inventory.CandidateCount != len(inventory.Documents) || inventory.PassedCount+inventory.ExcludedCount != inventory.CandidateCount {
		return errors.New("scan inventory is inconsistent")
	}
	return atomicWriteYAML(filePath, inventory)
}
