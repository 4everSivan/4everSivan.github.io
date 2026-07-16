package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

const (
	ManifestVersion = 2
	ManifestPath    = ".content-manifest.yaml"
)

type Manifest struct {
	Version          int                `yaml:"version"`
	GitleaksVersion  string             `yaml:"gitleaks_version"`
	RulesetSHA256    string             `yaml:"ruleset_sha256"`
	Documents        []ManifestDocument `yaml:"documents"`
	GeneratedIndexes []ManifestFile     `yaml:"generated_indexes"`
}

type ManifestDocument struct {
	Path          string   `yaml:"path"`
	SourceSHA256  string   `yaml:"source_sha256"`
	OutputSHA256  string   `yaml:"output_sha256"`
	ApprovedRules []string `yaml:"approved_rules,omitempty"`
}

type ManifestFile struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}

// EncodeManifest validates and deterministically encodes a controlled snapshot manifest.
func EncodeManifest(manifest Manifest) ([]byte, error) {
	if err := validateManifest(&manifest, true); err != nil {
		return nil, err
	}
	return yaml.Marshal(manifest)
}

// LoadManifest strictly parses a manifest and rejects unknown fields.
func LoadManifest(filename string) (Manifest, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Manifest{}, fmt.Errorf("read content manifest: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse content manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return Manifest{}, errors.New("content manifest contains multiple YAML documents")
	} else if !errors.Is(err, io.EOF) {
		return Manifest{}, fmt.Errorf("parse trailing content manifest data: %w", err)
	}
	if err := validateManifest(&manifest, false); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// VerifyManifest proves that a snapshot contains exactly the files and hashes
// declared by its manifest, with no untracked content.
func VerifyManifest(root string, manifest Manifest) error {
	if err := validateManifest(&manifest, false); err != nil {
		return err
	}
	actual, err := HashTree(root)
	if err != nil {
		return err
	}
	if _, ok := actual[ManifestPath]; !ok {
		return errors.New("content manifest is missing from snapshot")
	}
	delete(actual, ManifestPath)

	expected := make(map[string]string, len(manifest.Documents)+len(manifest.GeneratedIndexes))
	for _, document := range manifest.Documents {
		expected[document.Path] = document.OutputSHA256
	}
	for _, index := range manifest.GeneratedIndexes {
		expected[index.Path] = index.SHA256
	}
	if !equalHashes(expected, actual) {
		return errors.New("controlled snapshot does not match its manifest")
	}
	return nil
}

// ConfigFingerprint hashes versioned scanner inputs with path delimiters so a
// rules or approval change invalidates the current snapshot manifest.
func ConfigFingerprint(root string, relativePaths ...string) (string, error) {
	paths := append([]string(nil), relativePaths...)
	sort.Strings(paths)
	digest := sha256.New()
	for _, relative := range paths {
		if err := validatePath(relative); err != nil {
			return "", err
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", fmt.Errorf("read scanner configuration %q: %w", relative, err)
		}
		digest.Write([]byte(relative))
		digest.Write([]byte{0})
		digest.Write(data)
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func validateManifest(manifest *Manifest, normalizeOrder bool) error {
	if manifest.Version != ManifestVersion {
		return fmt.Errorf("unsupported content manifest version %d", manifest.Version)
	}
	if strings.TrimSpace(manifest.GitleaksVersion) == "" {
		return errors.New("content manifest has no Gitleaks version")
	}
	if !validSHA256(manifest.RulesetSHA256) {
		return errors.New("content manifest has invalid ruleset SHA-256")
	}
	if normalizeOrder {
		sort.Slice(manifest.Documents, func(i, j int) bool { return manifest.Documents[i].Path < manifest.Documents[j].Path })
		sort.Slice(manifest.GeneratedIndexes, func(i, j int) bool { return manifest.GeneratedIndexes[i].Path < manifest.GeneratedIndexes[j].Path })
	}

	seen := make(map[string]struct{}, len(manifest.Documents)+len(manifest.GeneratedIndexes))
	previous := ""
	for index := range manifest.Documents {
		document := &manifest.Documents[index]
		if err := validatePath(document.Path); err != nil || !validVisiblePath(document.Path) || filepath.Ext(document.Path) != ".md" || filepath.Base(document.Path) == "_index.md" {
			return fmt.Errorf("invalid manifest document path %q", document.Path)
		}
		if !normalizeOrder && previous != "" && document.Path <= previous {
			return errors.New("manifest documents are not strictly sorted")
		}
		previous = document.Path
		if !validSHA256(document.SourceSHA256) || !validSHA256(document.OutputSHA256) {
			return fmt.Errorf("invalid document hash for %q", document.Path)
		}
		if _, ok := seen[document.Path]; ok {
			return fmt.Errorf("duplicate manifest path %q", document.Path)
		}
		seen[document.Path] = struct{}{}
		if normalizeOrder {
			sort.Strings(document.ApprovedRules)
		}
		for ruleIndex, rule := range document.ApprovedRules {
			if strings.TrimSpace(rule) == "" || (ruleIndex > 0 && rule <= document.ApprovedRules[ruleIndex-1]) {
				return fmt.Errorf("invalid approved rule list for %q", document.Path)
			}
		}
	}
	previous = ""
	for _, index := range manifest.GeneratedIndexes {
		if err := validatePath(index.Path); err != nil || !validVisiblePath(index.Path) || filepath.Base(index.Path) != "_index.md" {
			return fmt.Errorf("invalid generated index path %q", index.Path)
		}
		if !normalizeOrder && previous != "" && index.Path <= previous {
			return errors.New("manifest indexes are not strictly sorted")
		}
		previous = index.Path
		if !validSHA256(index.SHA256) {
			return fmt.Errorf("invalid generated index hash for %q", index.Path)
		}
		if _, ok := seen[index.Path]; ok {
			return fmt.Errorf("duplicate manifest path %q", index.Path)
		}
		seen[index.Path] = struct{}{}
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func validVisiblePath(relative string) bool {
	for _, component := range strings.Split(relative, "/") {
		if component == "" || strings.HasPrefix(component, ".") {
			return false
		}
		for _, character := range component {
			if unicode.IsControl(character) {
				return false
			}
		}
	}
	return true
}
