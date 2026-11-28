package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newNotesRegistry(t *testing.T) (*Registry, *Root, string) {
	t.Helper()
	root, notes, _ := newTestRoot(t)
	return NewNotesRegistry(root), root, notes
}

func TestListFilesSortedWithSubfolders(t *testing.T) {
	reg, _, notes := newNotesRegistry(t)
	writeFile(t, filepath.Join(notes, "work", "b.md"), "beta")
	writeFile(t, filepath.Join(notes, "ignore.txt"), "not markdown")

	got, err := reg.Call("list_files", "{}")
	if err != nil {
		t.Fatal(err)
	}
	if got != "a.md\nwork/b.md" {
		t.Fatalf("got %q", got)
	}
}

func TestListFilesEmpty(t *testing.T) {
	root, err := NewRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewNotesRegistry(root).Call("list_files", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "No markdown files found") {
		t.Fatalf("got %q", got)
	}
}

func TestReadFile(t *testing.T) {
	reg, _, _ := newNotesRegistry(t)

	got, err := reg.Call("read_file", `{"path":"a.md"}`)
	if err != nil || got != "alpha" {
		t.Fatalf("got %q, %v", got, err)
	}

	_, err = reg.Call("read_file", `{"path":"missing.md"}`)
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("want not found error, got %v", err)
	}
}

func TestReadFileRejectsOutsideAndNonMarkdown(t *testing.T) {
	reg, _, notes := newNotesRegistry(t)
	writeFile(t, filepath.Join(notes, "data.txt"), "plain")
	outside := filepath.Join(filepath.Dir(notes), "secret.md")
	writeFile(t, outside, "top secret")

	if _, err := reg.Call("read_file", `{"path":"../secret.md"}`); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("traversal: got %v", err)
	}
	if _, err := reg.Call("read_file", `{"path":"../secret.txt"}`); err == nil {
		t.Fatal("non-markdown outside file must be rejected")
	}
	abs := strings.ReplaceAll(outside, `\`, `\\`)
	if got, err := reg.Call("read_file", `{"path":"`+abs+`"}`); !errors.Is(err, ErrOutsideRoot) || strings.Contains(got, "top secret") {
		t.Fatalf("absolute path: got %q, %v", got, err)
	}
	if _, err := reg.Call("read_file", `{"path":"data.txt"}`); err == nil {
		t.Fatal("non-markdown file must be rejected")
	}
}

func TestReadFileTruncates(t *testing.T) {
	reg, _, notes := newNotesRegistry(t)
	writeFile(t, filepath.Join(notes, "big.md"), strings.Repeat("x", MaxReadBytes+500))

	got, err := reg.Call("read_file", `{"path":"big.md"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "[truncated") {
		t.Fatal("missing truncation note")
	}
	if len(got) > MaxReadBytes+200 {
		t.Fatalf("output too long: %d", len(got))
	}
}

func TestReadFileTruncationKeepsValidUTF8(t *testing.T) {
	reg, _, notes := newNotesRegistry(t)
	// 3-byte runes so the byte limit lands mid-rune.
	writeFile(t, filepath.Join(notes, "uni.md"), strings.Repeat("€", MaxReadBytes))

	got, err := reg.Call("read_file", `{"path":"uni.md"}`)
	if err != nil {
		t.Fatal(err)
	}
	body := got[:strings.Index(got, "\n[truncated")]
	if strings.ToValidUTF8(body, "") != body {
		t.Fatal("truncated output is not valid UTF-8")
	}
}

func TestSearchTextCaseInsensitiveWithLineNumbers(t *testing.T) {
	reg, _, notes := newNotesRegistry(t)
	writeFile(t, filepath.Join(notes, "work", "b.md"), "first\nThe Deadline is Friday\n")
	writeFile(t, filepath.Join(notes, "c.md"), "no deadline set")

	got, err := reg.Call("search_text", `{"query":"DEADLINE"}`)
	if err != nil {
		t.Fatal(err)
	}
	want := "c.md:1: no deadline set\nwork/b.md:2: The Deadline is Friday"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSearchTextNoMatchAndEmptyQuery(t *testing.T) {
	reg, _, _ := newNotesRegistry(t)

	got, err := reg.Call("search_text", `{"query":"zzzz"}`)
	if err != nil || !strings.Contains(got, "No matches") {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := reg.Call("search_text", `{"query":"  "}`); err == nil {
		t.Fatal("empty query must fail")
	}
}

func TestSearchTextLimitsMatches(t *testing.T) {
	reg, _, notes := newNotesRegistry(t)
	writeFile(t, filepath.Join(notes, "many.md"), strings.Repeat("hit\n", MaxMatches+10))

	got, err := reg.Call("search_text", `{"query":"hit"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "more matches exist") {
		t.Fatal("missing truncation note")
	}
	if n := strings.Count(got, "many.md:"); n != MaxMatches {
		t.Fatalf("matches = %d, want %d", n, MaxMatches)
	}
}

func TestSearchSkipsSymlinkOutsideRoot(t *testing.T) {
	reg, _, notes := newNotesRegistry(t)
	outside := filepath.Join(filepath.Dir(notes), "outside.md")
	writeFile(t, outside, "needle in outside file")
	if err := os.Symlink(outside, filepath.Join(notes, "link.md")); err != nil {
		t.Skipf("cannot create symlink on this system: %v", err)
	}

	got, err := reg.Call("search_text", `{"query":"needle"}`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "needle") {
		t.Fatalf("search followed a symlink out of the root: %q", got)
	}
	if _, err := reg.Call("read_file", `{"path":"link.md"}`); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("read_file via symlink: got %v", err)
	}
}

func TestInvalidArguments(t *testing.T) {
	reg, _, _ := newNotesRegistry(t)
	if _, err := reg.Call("read_file", `{not json`); err == nil {
		t.Fatal("want error for malformed JSON")
	}
	if _, err := reg.Call("read_file", `{"path": 5}`); err == nil {
		t.Fatal("want error for wrong argument type")
	}
}

func TestDefinitionsAndUnknownTool(t *testing.T) {
	reg, _, _ := newNotesRegistry(t)

	defs := reg.Definitions()
	var names []string
	for _, d := range defs {
		names = append(names, d.Function.Name)
		if d.Type != "function" || d.Function.Description == "" || len(d.Function.Parameters) == 0 {
			t.Errorf("incomplete definition: %+v", d)
		}
	}
	if strings.Join(names, ",") != "list_files,read_file,search_text" {
		t.Fatalf("names = %v", names)
	}

	if _, err := reg.Call("delete_everything", "{}"); err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("want unknown tool error, got %v", err)
	}
}

func TestSampleCorpus(t *testing.T) {
	root, err := NewRoot(filepath.Join("..", "..", "testdata", "notes"))
	if err != nil {
		t.Fatal(err)
	}
	reg := NewNotesRegistry(root)

	files, err := reg.Call("list_files", "{}")
	if err != nil || !strings.Contains(files, "personal/recipes/lentil-soup.md") {
		t.Fatalf("list: %q, %v", files, err)
	}
	hit, err := reg.Call("search_text", `{"query":"PostgreSQL"}`)
	if err != nil || !strings.Contains(hit, "work/decisions.md") {
		t.Fatalf("search: %q, %v", hit, err)
	}
}
