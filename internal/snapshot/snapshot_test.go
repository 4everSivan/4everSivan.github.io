package snapshot

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReplaceCreatesCompleteSnapshotAndRemovesStaleFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	destination := filepath.Join(root, "docs")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "stale.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := []File{
		{Path: "_index.md", Data: []byte("index")},
		{Path: "分类/文档.md", Data: []byte("document")},
	}
	if err := Replace(destination, files); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "stale.md")); !os.IsNotExist(err) {
		t.Fatalf("stale file still exists: %v", err)
	}
	want := map[string]string{
		"_index.md": "1bc04b5291c26a46d918139138b992d2de976d6851d0893b0476b85bfbdfc6e6",
		"分类/文档.md":  "43cc23fa52b87b4cc1d02b5b114154151d6adddb17c9fddc06b027fa99e24008",
	}
	got, err := HashTree(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hash tree mismatch: got %v, want %v", got, want)
	}
}

func TestInvalidReplacementPreservesPreviousSnapshot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	destination := filepath.Join(root, "docs")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := filepath.Join(destination, "kept.md")
	if err := os.WriteFile(previous, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Replace(destination, []File{{Path: "../escape.md", Data: []byte("bad")}})
	if err == nil {
		t.Fatal("expected invalid path error")
	}
	got, readErr := os.ReadFile(previous)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "keep" {
		t.Fatalf("previous snapshot changed: %q", got)
	}
}

func TestReplaceIsIdempotent(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "docs")
	files := []File{{Path: "x.md", Data: []byte("same")}}
	if err := Replace(destination, files); err != nil {
		t.Fatal(err)
	}
	first, err := HashTree(destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := Replace(destination, files); err != nil {
		t.Fatal(err)
	}
	second, err := HashTree(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("idempotent snapshot changed: %v != %v", first, second)
	}
}

func TestFinalValidationFailurePreservesPreviousSnapshot(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "docs")
	if err := Replace(destination, []File{{Path: "old.md", Data: []byte("old")}}); err != nil {
		t.Fatal(err)
	}
	err := ReplaceValidated(destination, []File{{Path: "new.md", Data: []byte("new")}}, func() error {
		return errors.New("source changed")
	})
	if err == nil {
		t.Fatal("expected validation failure")
	}
	if _, err := os.Stat(filepath.Join(destination, "old.md")); err != nil {
		t.Fatalf("previous snapshot was not preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "new.md")); !os.IsNotExist(err) {
		t.Fatalf("new snapshot was activated: %v", err)
	}
}

func TestAtomicExchangeFailurePreservesPreviousSnapshot(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "docs")
	if err := Replace(destination, []File{{Path: "old.md", Data: []byte("old")}}); err != nil {
		t.Fatal(err)
	}
	err := replaceValidated(destination, []File{{Path: "new.md", Data: []byte("new")}}, nil, func(_, _ string) error {
		return errors.New("synthetic exchange failure")
	})
	if err == nil {
		t.Fatal("expected exchange failure")
	}
	if data, readErr := os.ReadFile(filepath.Join(destination, "old.md")); readErr != nil || string(data) != "old" {
		t.Fatalf("previous snapshot changed after failed exchange: data=%q err=%v", data, readErr)
	}
	if _, err := os.Stat(filepath.Join(destination, "new.md")); !os.IsNotExist(err) {
		t.Fatalf("new snapshot became visible after failed exchange: %v", err)
	}
}

func TestReplaceRejectsSymlinkParentWithoutWritingOutside(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	external := t.TempDir()
	content := filepath.Join(root, "content")
	if err := os.Symlink(external, content); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := Replace(filepath.Join(content, "docs"), []File{{Path: "new.md", Data: []byte("new")}})
	if err == nil {
		t.Fatal("expected symlink parent rejection")
	}
	if _, statErr := os.Lstat(filepath.Join(external, "docs")); !os.IsNotExist(statErr) {
		t.Fatalf("snapshot wrote through symlink parent: %v", statErr)
	}
}

func TestReplaceRejectsNonDirectoryDestination(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "docs")
	if err := os.WriteFile(destination, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Replace(destination, []File{{Path: "new.md", Data: []byte("new")}}); err == nil {
		t.Fatal("expected non-directory destination rejection")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "not a directory" {
		t.Fatalf("non-directory destination changed: data=%q err=%v", data, err)
	}
}

func TestRecoversInterruptedLegacyBackupBeforeReplacement(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	destination := filepath.Join(root, "docs")
	backup := destination + ".previous"
	if err := os.MkdirAll(backup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backup, "old.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Replace(destination, []File{{Path: "new.md", Data: []byte("new")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "new.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("legacy backup survived recovery: %v", err)
	}
}
