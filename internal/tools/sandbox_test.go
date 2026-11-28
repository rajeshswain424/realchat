package tools

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// newTestRoot creates <tmp>/notes (the root) and <tmp>/secret.txt (outside it).
func newTestRoot(t *testing.T) (root *Root, notesDir, outsideFile string) {
	t.Helper()
	tmp := t.TempDir()
	notesDir = filepath.Join(tmp, "notes")
	if err := os.MkdirAll(filepath.Join(notesDir, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	outsideFile = filepath.Join(tmp, "secret.txt")
	writeFile(t, outsideFile, "top secret")
	writeFile(t, filepath.Join(notesDir, "a.md"), "alpha")

	root, err := NewRoot(notesDir)
	if err != nil {
		t.Fatal(err)
	}
	return root, notesDir, outsideFile
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAllowsFilesInsideRoot(t *testing.T) {
	root, _, _ := newTestRoot(t)
	writeFile(t, filepath.Join(root.dir, "work", "b.md"), "beta")

	for _, rel := range []string{"a.md", "work/b.md", "./a.md", "work/../a.md"} {
		if _, err := root.Resolve(rel); err != nil {
			t.Errorf("Resolve(%q) = %v, want ok", rel, err)
		}
	}
}

func TestResolveRejectsEscapes(t *testing.T) {
	root, _, outside := newTestRoot(t)

	cases := map[string]string{
		"parent":         "../secret.txt",
		"deep parent":    "work/../../secret.txt",
		"backslash up":   `..\secret.txt`,
		"absolute":       outside,
		"leading slash":  "/etc/passwd",
		"leading bslash": `\windows\win.ini`,
		"drive letter":   `C:\Windows\win.ini`,
		"drive relative": `C:secret.txt`,
		"only parent":    "..",
	}
	for name, rel := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := root.Resolve(rel)
			if !errors.Is(err, ErrOutsideRoot) {
				t.Fatalf("Resolve(%q) = %v, want ErrOutsideRoot", rel, err)
			}
		})
	}
}

func TestResolveMissingFile(t *testing.T) {
	root, _, _ := newTestRoot(t)
	if _, err := root.Resolve("nope.md"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("want ErrNotExist, got %v", err)
	}
}

func TestResolveRejectsEmptyPath(t *testing.T) {
	root, _, _ := newTestRoot(t)
	if _, err := root.Resolve(""); err == nil {
		t.Fatal("want error for empty path")
	}
}

func TestResolveRejectsSymlinkOutside(t *testing.T) {
	root, notes, outside := newTestRoot(t)
	if err := os.Symlink(outside, filepath.Join(notes, "link.md")); err != nil {
		t.Skipf("cannot create symlink on this system: %v", err)
	}
	if _, err := root.Resolve("link.md"); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("want ErrOutsideRoot, got %v", err)
	}
}

func TestNewRootMissingFolder(t *testing.T) {
	if _, err := NewRoot(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("want error for missing folder")
	}
}
