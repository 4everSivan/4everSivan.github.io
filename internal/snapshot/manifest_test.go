package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestRoundTripAndVerification(t *testing.T) {
	t.Parallel()
	document := []byte("document")
	index := []byte("index")
	documentHash := hash(document)
	indexHash := hash(index)
	manifest := Manifest{
		Version:         ManifestVersion,
		GitleaksVersion: "8.30.1",
		RulesetSHA256:   hash([]byte("rules")),
		Documents: []ManifestDocument{{
			Path: "分类/文档.md", SourceSHA256: hash([]byte("source")), OutputSHA256: documentHash,
		}},
		GeneratedIndexes: []ManifestFile{{Path: "_index.md", SHA256: indexHash}},
	}
	encoded, err := EncodeManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	files := []File{
		{Path: "分类/文档.md", Data: document},
		{Path: "_index.md", Data: index},
		{Path: ManifestPath, Data: encoded},
	}
	if err := Replace(root, files); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadManifest(filepath.Join(root, ManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyManifest(root, loaded); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyManifestRejectsExtraFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manifest := Manifest{
		Version:          ManifestVersion,
		GitleaksVersion:  "8.30.1",
		RulesetSHA256:    hash([]byte("rules")),
		GeneratedIndexes: []ManifestFile{{Path: "_index.md", SHA256: hash([]byte("index"))}},
	}
	encoded, err := EncodeManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ManifestPath), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_index.md"), []byte("index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "extra.md"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyManifest(root, manifest); err == nil {
		t.Fatal("expected extra file to fail verification")
	}
}

func TestLoadManifestRejectsTrailingDocument(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	filename := filepath.Join(root, ManifestPath)
	data := []byte("version: 2\ngitleaks_version: 8.30.1\nruleset_sha256: " + hash([]byte("rules")) + "\ndocuments: []\ngenerated_indexes: []\n---\nextra: true\n")
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(filename); err == nil {
		t.Fatal("LoadManifest accepted a second YAML document")
	}
}

func TestEncodeManifestRejectsHiddenAndControlPaths(t *testing.T) {
	t.Parallel()
	for _, invalidPath := range []string{".hidden/note.md", "folder/bad\nname.md"} {
		manifest := Manifest{
			Version:         ManifestVersion,
			GitleaksVersion: "8.30.1",
			RulesetSHA256:   hash([]byte("rules")),
			Documents: []ManifestDocument{{
				Path: invalidPath, SourceSHA256: hash([]byte("source")), OutputSHA256: hash([]byte("output")),
			}},
		}
		if _, err := EncodeManifest(manifest); err == nil {
			t.Fatalf("EncodeManifest accepted invalid path %q", invalidPath)
		}
	}
}

func hash(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
