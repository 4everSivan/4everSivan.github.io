package source

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"syscall"
	"testing"
	"time"
)

func TestDiscoverFiltersAndSortsCandidates(t *testing.T) {
	root := t.TempDir()
	writeSynthetic(t, filepath.Join(root, "分类", "有 空格.md"), "# 合成标题\n")
	writeSynthetic(t, filepath.Join(root, "alpha.md"), "# Alpha\n")
	writeSynthetic(t, filepath.Join(root, "UPPER.MD"), "not eligible")
	writeSynthetic(t, filepath.Join(root, "note.txt"), "not markdown")
	writeSynthetic(t, filepath.Join(root, ".hidden.md"), "hidden")
	writeSynthetic(t, filepath.Join(root, "draft.tmp.md"), "temporary")
	writeSynthetic(t, filepath.Join(root, ".private", "inside.md"), "hidden directory")

	manifest, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	got := make([]string, 0, len(manifest.Candidates))
	for _, candidate := range manifest.Candidates {
		got = append(got, candidate.RelativePath)
		if len(candidate.SHA256) != 64 {
			t.Errorf("candidate %q SHA-256 length = %d", candidate.RelativePath, len(candidate.SHA256))
		}
	}
	want := []string{"alpha.md", "分类/有 空格.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate paths = %#v, want %#v", got, want)
	}
}

func TestDiscoverRejectsSymlinksAndSpecialFiles(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	writeSynthetic(t, outside, "outside")
	if err := os.Symlink(outside, filepath.Join(root, "escape.md")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	pipe := filepath.Join(root, "stream.md")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatalf("create named pipe: %v", err)
	}

	manifest, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(manifest.Candidates) != 0 {
		t.Fatalf("unexpected candidates: %#v", manifest.Candidates)
	}
	reasons := map[string]string{}
	for _, skipped := range manifest.Skipped {
		reasons[skipped.RelativePath] = skipped.Reason
	}
	if reasons["escape.md"] != "symbolic-link" {
		t.Errorf("symlink reason = %q", reasons["escape.md"])
	}
	if reasons["stream.md"] != "not-regular" {
		t.Errorf("special file reason = %q", reasons["stream.md"])
	}
}

func TestReadDetectsChangedCandidate(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	writeSynthetic(t, path, "first")
	manifest, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(manifest.Candidates) != 1 {
		t.Fatalf("candidate count = %d", len(manifest.Candidates))
	}

	// Ensure the metadata differs even on coarse timestamp filesystems.
	time.Sleep(time.Millisecond)
	writeSynthetic(t, path, "second")
	_, err = Read(manifest.Root, manifest.Candidates[0])
	if !errors.Is(err, ErrChanged) {
		t.Fatalf("Read() error = %v, want ErrChanged", err)
	}
}

func TestDiscoverDoesNotWriteSourceTree(t *testing.T) {
	root := t.TempDir()
	writeSynthetic(t, filepath.Join(root, "one.md"), "one")
	writeSynthetic(t, filepath.Join(root, "nested", "two.md"), "two")
	before := treeMetadata(t, root)

	if _, err := Discover(root); err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	after := treeMetadata(t, root)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("source tree changed:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestWithinRejectsSiblingPrefix(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "docs")
	sibling := filepath.Join(parent, "docs-private", "note.md")
	if within(root, sibling) {
		t.Fatal("within() accepted sibling with common string prefix")
	}
}

func writeSynthetic(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func treeMetadata(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, rel+"|"+info.Mode().String()+"|"+info.ModTime().UTC().Format(time.RFC3339Nano))
		return nil
	})
	if err != nil {
		t.Fatalf("walk metadata: %v", err)
	}
	sort.Strings(entries)
	return entries
}
