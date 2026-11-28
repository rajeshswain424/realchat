package tools

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"unicode/utf8"
)

const (
	MaxReadBytes   = 20000
	MaxListEntries = 500
	MaxMatches     = 50
	maxLineChars   = 300
)

func (r *Root) listFiles(json.RawMessage) (string, error) {
	files, err := r.markdownFiles()
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "No markdown files found.", nil
	}
	truncated := len(files) > MaxListEntries
	if truncated {
		files = files[:MaxListEntries]
	}
	out := strings.Join(files, "\n")
	if truncated {
		out += fmt.Sprintf("\n[truncated: showing the first %d files]", MaxListEntries)
	}
	return out, nil
}

func (r *Root) readFile(args json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if !strings.HasSuffix(strings.ToLower(in.Path), ".md") {
		return "", errors.New("only .md files can be read")
	}

	path, err := r.Resolve(in.Path)
	if err != nil {
		return "", pathError(in.Path, err)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", pathError(in.Path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, MaxReadBytes+1))
	if err != nil {
		return "", pathError(in.Path, err)
	}
	if len(data) <= MaxReadBytes {
		return string(data), nil
	}

	cut := MaxReadBytes
	for cut > 0 && !utf8.RuneStart(data[cut]) {
		cut--
	}
	return string(data[:cut]) + fmt.Sprintf("\n[truncated: showing the first %d bytes of the file]", cut), nil
}

func (r *Root) searchText(args json.RawMessage) (string, error) {
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	query := strings.ToLower(strings.TrimSpace(in.Query))
	if query == "" {
		return "", errors.New("query is required")
	}

	files, err := r.markdownFiles()
	if err != nil {
		return "", err
	}

	var matches []string
	more := false
scan:
	for _, rel := range files {
		path, err := r.Resolve(rel)
		if err != nil {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for n := 1; sc.Scan(); n++ {
			line := sc.Text()
			if !strings.Contains(strings.ToLower(line), query) {
				continue
			}
			if len(matches) == MaxMatches {
				more = true
				f.Close()
				break scan
			}
			matches = append(matches, fmt.Sprintf("%s:%d: %s", rel, n, clip(strings.TrimSpace(line))))
		}
		f.Close()
	}

	if len(matches) == 0 {
		return "No matches found.", nil
	}
	out := strings.Join(matches, "\n")
	if more {
		out += fmt.Sprintf("\n[truncated: more matches exist, showing the first %d]", MaxMatches)
	}
	return out, nil
}

func clip(s string) string {
	if utf8.RuneCountInString(s) <= maxLineChars {
		return s
	}
	return string([]rune(s)[:maxLineChars]) + "..."
}

// pathError turns filesystem errors into messages that are useful to the model.
func pathError(path string, err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("file not found: %s", path)
	case errors.Is(err, ErrOutsideRoot):
		return fmt.Errorf("%s: %w", path, ErrOutsideRoot)
	default:
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
}
