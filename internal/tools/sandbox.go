package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrOutsideRoot is returned for any path that would leave the notes folder.
var ErrOutsideRoot = errors.New("path is outside the notes folder")

// Root is the notes folder that every tool is confined to.
type Root struct {
	dir string // absolute, symlinks resolved
}

// NewRoot validates dir and resolves it to an absolute path.
func NewRoot(dir string) (*Root, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("notes folder %q not found", dir)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("notes folder %q is not a directory", dir)
	}
	return &Root{dir: resolved}, nil
}

// Resolve maps a slash-separated relative path to a real path inside the root.
// It rejects absolute paths, drive letters, ".." escapes and symlinks that lead outside.
func (r *Root) Resolve(rel string) (string, error) {
	if rel == "" {
		return "", errors.New("path is required")
	}
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return "", ErrOutsideRoot
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrOutsideRoot
	}

	resolved, err := filepath.EvalSymlinks(filepath.Join(r.dir, clean))
	if err != nil {
		return "", err
	}
	if !r.contains(resolved) {
		return "", ErrOutsideRoot
	}
	return resolved, nil
}

func (r *Root) contains(path string) bool {
	rel, err := filepath.Rel(r.dir, path)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// markdownFiles returns the slash-separated relative paths of all .md files, sorted.
// Symlinked directories are not followed, and symlinked files that leave the root are skipped.
func (r *Root) markdownFiles() ([]string, error) {
	var files []string
	err := filepath.WalkDir(r.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		rel, err := filepath.Rel(r.dir, path)
		if err != nil {
			return nil
		}
		resolved, err := r.Resolve(filepath.ToSlash(rel))
		if err != nil {
			return nil
		}
		if info, err := os.Stat(resolved); err != nil || !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
