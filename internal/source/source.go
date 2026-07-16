// Package source discovers immutable Markdown candidates below a configured
// read-only source root.
package source

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

const DefaultRoot = "/Users/sivan/work/学习文档"

// Candidate is a regular Markdown file whose real path is contained by Root.
// AbsolutePath is runtime-only input and must never be written to a public
// report.
type Candidate struct {
	AbsolutePath string    `json:"-" yaml:"-"`
	RelativePath string    `json:"relative_path" yaml:"relative_path"`
	SHA256       string    `json:"sha256" yaml:"sha256"`
	State        FileState `json:"-" yaml:"-"`
}

// FileState records enough information to detect both content and identity
// changes around a scan.
type FileState struct {
	Size            int64
	Mode            fs.FileMode
	ModTimeUnixNano int64
	Device          uint64
	Inode           uint64
}

// Skip explains why a path was not considered a publishable candidate. It
// never contains file contents.
type Skip struct {
	RelativePath string
	Reason       string
}

// Manifest is deterministic: Candidates and Skipped are sorted by relative
// slash-separated path.
type Manifest struct {
	Root       string
	Candidates []Candidate
	Skipped    []Skip
}

var ErrChanged = errors.New("source file changed during scan")

// Discover recursively enumerates lower-case .md regular files. It does not
// follow symbolic links and it reads content only after all name, type and
// containment checks have passed.
func Discover(root string) (Manifest, error) {
	if root == "" {
		return Manifest{}, errors.New("source root is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Manifest{}, fmt.Errorf("resolve source root: %w", err)
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return Manifest{}, fmt.Errorf("resolve source root symlinks: %w", err)
	}
	rootInfo, err := os.Stat(rootReal)
	if err != nil {
		return Manifest{}, fmt.Errorf("stat source root: %w", err)
	}
	if !rootInfo.IsDir() {
		return Manifest{}, errors.New("source root is not a directory")
	}

	manifest := Manifest{Root: rootReal}
	err = filepath.WalkDir(rootReal, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk source path: %w", walkErr)
		}
		if path == rootReal {
			return nil
		}

		rel, err := filepath.Rel(rootReal, path)
		if err != nil {
			return fmt.Errorf("derive source relative path: %w", err)
		}
		rel = filepath.ToSlash(rel)

		if entry.Type()&os.ModeSymlink != 0 {
			manifest.Skipped = append(manifest.Skipped, Skip{RelativePath: rel, Reason: "symbolic-link"})
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if isHiddenOrTemporary(rel) {
			manifest.Skipped = append(manifest.Skipped, Skip{RelativePath: rel, Reason: "hidden-or-temporary"})
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) != ".md" {
			manifest.Skipped = append(manifest.Skipped, Skip{RelativePath: rel, Reason: "not-markdown"})
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat candidate %q: %w", rel, err)
		}
		if !info.Mode().IsRegular() {
			manifest.Skipped = append(manifest.Skipped, Skip{RelativePath: rel, Reason: "not-regular"})
			return nil
		}

		real, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve candidate %q: %w", rel, err)
		}
		if !within(rootReal, real) {
			manifest.Skipped = append(manifest.Skipped, Skip{RelativePath: rel, Reason: "path-escape"})
			return nil
		}

		data, state, digest, err := readStable(rootReal, path)
		if err != nil {
			return fmt.Errorf("fingerprint candidate %q: %w", rel, err)
		}
		_ = data // Discovery retains only the fingerprint, never source content.
		manifest.Candidates = append(manifest.Candidates, Candidate{
			AbsolutePath: path,
			RelativePath: rel,
			SHA256:       digest,
			State:        state,
		})
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}

	sort.Slice(manifest.Candidates, func(i, j int) bool {
		return manifest.Candidates[i].RelativePath < manifest.Candidates[j].RelativePath
	})
	sort.Slice(manifest.Skipped, func(i, j int) bool {
		return manifest.Skipped[i].RelativePath < manifest.Skipped[j].RelativePath
	})
	return manifest, nil
}

// Read returns a candidate's bytes only if path, identity and content still
// match the discovery manifest.
func Read(root string, candidate Candidate) ([]byte, error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve source root: %w", err)
	}
	data, state, digest, err := readStable(rootReal, candidate.AbsolutePath)
	if err != nil {
		switch {
		case errors.Is(err, ErrChanged):
			return nil, fmt.Errorf("%w: %s", ErrChanged, candidate.RelativePath)
		case errors.Is(err, os.ErrNotExist):
			return nil, fmt.Errorf("source candidate disappeared: %s", candidate.RelativePath)
		case errors.Is(err, os.ErrPermission):
			return nil, fmt.Errorf("source candidate is not readable: %s", candidate.RelativePath)
		default:
			return nil, fmt.Errorf("source candidate validation failed: %s", candidate.RelativePath)
		}
	}
	if state != candidate.State || digest != candidate.SHA256 {
		return nil, fmt.Errorf("%w: %s", ErrChanged, candidate.RelativePath)
	}
	return data, nil
}

// Validate checks that a candidate has not changed since discovery.
func Validate(root string, candidate Candidate) error {
	_, err := Read(root, candidate)
	return err
}

func readStable(root, path string) ([]byte, FileState, string, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, FileState{}, "", err
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, FileState{}, "", errors.New("candidate is not a regular non-symlink file")
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, FileState{}, "", err
	}
	if !within(root, real) {
		return nil, FileState{}, "", errors.New("candidate real path escapes source root")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, FileState{}, "", err
	}
	opened, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, FileState{}, "", err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		_ = file.Close()
		return nil, FileState{}, "", ErrChanged
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, FileState{}, "", readErr
	}
	if closeErr != nil {
		return nil, FileState{}, "", closeErr
	}

	after, err := os.Lstat(path)
	if err != nil {
		return nil, FileState{}, "", err
	}
	beforeState := stateFromInfo(before)
	afterState := stateFromInfo(after)
	if beforeState != afterState || !after.Mode().IsRegular() || after.Mode()&os.ModeSymlink != 0 {
		return nil, FileState{}, "", ErrChanged
	}
	afterReal, err := filepath.EvalSymlinks(path)
	if err != nil || afterReal != real || !within(root, afterReal) {
		return nil, FileState{}, "", ErrChanged
	}

	sum := sha256.Sum256(data)
	return data, afterState, hex.EncodeToString(sum[:]), nil
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func isHiddenOrTemporary(relativePath string) bool {
	for _, part := range strings.Split(filepath.ToSlash(relativePath), "/") {
		lower := strings.ToLower(part)
		if strings.HasPrefix(part, ".") || strings.HasPrefix(part, "#") || strings.HasSuffix(part, "#") || strings.HasSuffix(part, "~") {
			return true
		}
		for _, marker := range []string{".tmp", ".temp", ".bak", ".backup", ".swp", ".swo"} {
			if strings.HasSuffix(lower, marker) || strings.Contains(lower, marker+".") {
				return true
			}
		}
	}
	return false
}
