package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// File is one file in a controlled content snapshot. Path always uses slash
// separators and is relative to the snapshot root.
type File struct {
	Path string
	Data []byte
}

// Replace builds a complete sibling directory and swaps it into destination.
// Validation and writes complete before the current snapshot is touched.
func Replace(destination string, files []File) error {
	return ReplaceValidated(destination, files, nil)
}

// ReplaceValidated runs finalValidation after the complete temporary snapshot
// has passed its integrity check and immediately before the current snapshot
// is touched. This is used to recheck source identities at the commit point.
func ReplaceValidated(destination string, files []File, finalValidation func() error) error {
	return replaceValidated(destination, files, finalValidation, atomicExchangeDirectories)
}

func replaceValidated(destination string, files []File, finalValidation func() error, exchange func(string, string) error) error {
	if destination == "" {
		return errors.New("snapshot destination is empty")
	}
	if info, err := os.Lstat(destination); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("snapshot destination must not be a symlink")
		}
		if !info.IsDir() {
			return errors.New("snapshot destination must be a directory")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect snapshot destination: %w", err)
	}

	normalized, expected, err := normalize(files)
	if err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create snapshot parent: %w", err)
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		return fmt.Errorf("inspect snapshot parent: %w", err)
	}
	if parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
		return errors.New("snapshot parent must be a real directory")
	}
	if err := recoverLegacyBackup(destination); err != nil {
		return err
	}
	if err := cleanupStaleTemporarySnapshots(parent, filepath.Base(destination)); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(parent, "."+filepath.Base(destination)+"-tmp-")
	if err != nil {
		return fmt.Errorf("create temporary snapshot: %w", err)
	}
	defer os.RemoveAll(temporary)

	for _, file := range normalized {
		target := filepath.Join(temporary, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create snapshot directory for %q: %w", file.Path, err)
		}
		if err := os.WriteFile(target, file.Data, 0o644); err != nil {
			return fmt.Errorf("write snapshot file %q: %w", file.Path, err)
		}
	}

	actual, err := HashTree(temporary)
	if err != nil {
		return fmt.Errorf("verify temporary snapshot: %w", err)
	}
	if !equalHashes(expected, actual) {
		return errors.New("temporary snapshot integrity check failed")
	}
	if err := syncTree(temporary); err != nil {
		return fmt.Errorf("sync temporary snapshot: %w", err)
	}
	if finalValidation != nil {
		if err := finalValidation(); err != nil {
			return fmt.Errorf("final snapshot validation: %w", err)
		}
	}

	if _, err := os.Lstat(destination); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(temporary, destination); err != nil {
			return fmt.Errorf("activate initial snapshot: %w", err)
		}
		if err := syncDirectory(parent); err != nil {
			if restoreErr := os.Rename(destination, temporary); restoreErr != nil {
				return fmt.Errorf("sync initial snapshot parent: %v; undo activation: %w", err, restoreErr)
			}
			return fmt.Errorf("sync initial snapshot parent: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect current snapshot: %w", err)
	}

	if err := exchange(destination, temporary); err != nil {
		return fmt.Errorf("atomically exchange content snapshot: %w", err)
	}
	if err := syncDirectory(parent); err != nil {
		if restoreErr := exchange(destination, temporary); restoreErr != nil {
			return fmt.Errorf("sync activated snapshot parent: %v; restore previous snapshot: %w", err, restoreErr)
		}
		return fmt.Errorf("sync activated snapshot parent: %w", err)
	}
	return nil
}

func recoverLegacyBackup(destination string) error {
	backup := destination + ".previous"
	backupInfo, backupErr := os.Lstat(backup)
	if errors.Is(backupErr, os.ErrNotExist) {
		return nil
	}
	if backupErr != nil {
		return fmt.Errorf("inspect legacy snapshot backup: %w", backupErr)
	}
	if !backupInfo.IsDir() || backupInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("legacy snapshot backup is not a regular directory")
	}
	if _, err := os.Lstat(destination); err == nil {
		return errors.New("both current snapshot and legacy backup exist; refusing ambiguous recovery")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect current snapshot during recovery: %w", err)
	}
	if err := os.Rename(backup, destination); err != nil {
		return fmt.Errorf("restore interrupted legacy snapshot: %w", err)
	}
	if err := syncDirectory(filepath.Dir(destination)); err != nil {
		return fmt.Errorf("sync restored legacy snapshot: %w", err)
	}
	return nil
}

func syncDirectory(directory string) error {
	handle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Sync()
}

func syncTree(root string) error {
	directories := make([]string, 0)
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			directories = append(directories, current)
			return nil
		}
		file, err := os.Open(current)
		if err != nil {
			return err
		}
		syncErr := file.Sync()
		closeErr := file.Close()
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	})
	if err != nil {
		return err
	}
	sort.Slice(directories, func(i, j int) bool {
		return strings.Count(directories[i], string(filepath.Separator)) > strings.Count(directories[j], string(filepath.Separator))
	})
	for _, directory := range directories {
		if err := syncDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

func cleanupStaleTemporarySnapshots(parent, destinationBase string) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return fmt.Errorf("list snapshot parent: %w", err)
	}
	prefix := "." + destinationBase + "-tmp-"
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("stale snapshot temporary path is not a regular directory")
		}
		if err := os.RemoveAll(filepath.Join(parent, entry.Name())); err != nil {
			return fmt.Errorf("remove stale snapshot temporary directory: %w", err)
		}
	}
	return nil
}

// HashTree returns deterministic SHA-256 values for every regular file.
func HashTree(root string) (map[string]string, error) {
	hashes := make(map[string]string)
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("snapshot contains symlink %q", current)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("snapshot contains non-regular file %q", current)
		}
		file, err := os.Open(current)
		if err != nil {
			return err
		}
		digest := sha256.New()
		_, copyErr := io.Copy(digest, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		hashes[filepath.ToSlash(relative)] = hex.EncodeToString(digest.Sum(nil))
		return nil
	})
	return hashes, err
}

func normalize(files []File) ([]File, map[string]string, error) {
	normalized := append([]File(nil), files...)
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Path < normalized[j].Path })
	hashes := make(map[string]string, len(normalized))
	for _, file := range normalized {
		if err := validatePath(file.Path); err != nil {
			return nil, nil, err
		}
		if _, exists := hashes[file.Path]; exists {
			return nil, nil, fmt.Errorf("duplicate snapshot path %q", file.Path)
		}
		digest := sha256.Sum256(file.Data)
		hashes[file.Path] = hex.EncodeToString(digest[:])
	}
	return normalized, hashes, nil
}

func validatePath(relative string) error {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "\\") {
		return fmt.Errorf("invalid snapshot path %q", relative)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative)))
	if clean != relative || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("invalid snapshot path %q", relative)
	}
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid snapshot path %q", relative)
		}
	}
	return nil
}

func equalHashes(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for name, hash := range left {
		if right[name] != hash {
			return false
		}
	}
	return true
}
