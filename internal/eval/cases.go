package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Case is one question with the checks its answer must satisfy.
type Case struct {
	Question       string   `json:"question"`
	ExpectContains []string `json:"expect_contains"`
	ExpectTool     string   `json:"expect_tool,omitempty"`
}

// LoadCases reads and validates an eval file. Unknown fields are rejected so typos do not pass silently.
func LoadCases(path string) ([]Case, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open eval file: %w", err)
	}
	defer f.Close()

	var cases []Case
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cases); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("%s contains no cases", path)
	}
	for i, c := range cases {
		if strings.TrimSpace(c.Question) == "" {
			return nil, fmt.Errorf("%s: case %d has an empty question", path, i+1)
		}
		if len(c.ExpectContains) == 0 {
			return nil, fmt.Errorf("%s: case %d (%q) has no expect_contains", path, i+1, c.Question)
		}
		for _, want := range c.ExpectContains {
			if strings.TrimSpace(want) == "" {
				return nil, fmt.Errorf("%s: case %d (%q) has an empty expect_contains entry", path, i+1, c.Question)
			}
		}
	}
	return cases, nil
}

// Check reports whether an answer satisfies a case, and why not when it does not.
func Check(c Case, answer string, toolsCalled []string) (bool, string) {
	got := normalize(answer)
	for _, want := range c.ExpectContains {
		if !strings.Contains(got, normalize(want)) {
			return false, fmt.Sprintf("answer does not contain %q", want)
		}
	}
	if c.ExpectTool != "" {
		for _, name := range toolsCalled {
			if name == c.ExpectTool {
				return true, ""
			}
		}
		return false, fmt.Sprintf("required tool %q was not called", c.ExpectTool)
	}
	return true, ""
}

// normalize lowercases and maps the typographic spaces and hyphens models like to emit onto plain ones.
func normalize(s string) string {
	r := strings.NewReplacer(
		"\u00a0", " ", "\u202f", " ", "\u2009", " ",
		"\u2010", "-", "\u2011", "-", "\u2013", "-",
	)
	return strings.Join(strings.Fields(strings.ToLower(r.Replace(s))), " ")
}
