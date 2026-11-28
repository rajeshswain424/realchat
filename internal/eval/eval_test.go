package eval

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"realchat/internal/agent"
	"realchat/internal/llm"
)

func writeCases(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "eval.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadCasesValid(t *testing.T) {
	cases, err := LoadCases(writeCases(t, `[{"question":"q","expect_contains":["a"],"expect_tool":"read_file"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].ExpectTool != "read_file" {
		t.Fatalf("cases = %+v", cases)
	}
}

func TestLoadCasesRejectsMalformed(t *testing.T) {
	bad := map[string]string{
		"empty question":  `[{"question":" ","expect_contains":["a"]}]`,
		"no expectations": `[{"question":"q"}]`,
		"empty expected":  `[{"question":"q","expect_contains":[""]}]`,
		"unknown field":   `[{"question":"q","expect_contains":["a"],"expect_tols":"x"}]`,
		"not json":        `{nope`,
		"no cases":        `[]`,
	}
	for name, content := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadCases(writeCases(t, content)); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestLoadCasesMissingFile(t *testing.T) {
	if _, err := LoadCases(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("want error")
	}
}

func TestBundledEvalFileIsValid(t *testing.T) {
	cases, err := LoadCases(filepath.Join("..", "..", "testdata", "eval.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 10 || len(cases) > 15 {
		t.Fatalf("expected 10 to 15 cases, got %d", len(cases))
	}
}

func TestCheck(t *testing.T) {
	c := Case{Question: "q", ExpectContains: []string{"100", "per minute"}, ExpectTool: "search_text"}

	if ok, reason := Check(c, "The limit is 100 requests PER MINUTE.", []string{"search_text"}); !ok {
		t.Fatalf("pass case failed: %s", reason)
	}

	ok, reason := Check(c, "I do not know", []string{"search_text"})
	if ok || !strings.Contains(reason, `"100"`) {
		t.Fatalf("wrong answer: %v %q", ok, reason)
	}

	ok, reason = Check(c, "100 requests per minute", []string{"read_file"})
	if ok || !strings.Contains(reason, "search_text") {
		t.Fatalf("missing tool: %v %q", ok, reason)
	}
}

func TestCheckNormalizesTypography(t *testing.T) {
	c := Case{ExpectContains: []string{"15 minutes", "garden/bed1"}}
	if ok, reason := Check(c, "Every 15\u202fminutes on garden/bed1", nil); !ok {
		t.Fatalf("narrow nbsp not normalized: %s", reason)
	}
}

// scriptedModel answers by question and can be told to fail for one.
type scriptedModel struct {
	answers   map[string]string
	failOn    string
	callTools map[string]string // question -> tool to call first
}

func (s *scriptedModel) Chat(_ context.Context, msgs []llm.Message, _ []llm.ToolDefinition) (llm.Message, error) {
	var question string
	for _, m := range msgs {
		if m.Role == llm.RoleUser {
			question = m.Content
		}
	}
	if question == s.failOn {
		return llm.Message{}, errors.New("network down")
	}
	if tool, ok := s.callTools[question]; ok && msgs[len(msgs)-1].Role != llm.RoleTool {
		return llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{
			{ID: "c1", Type: "function", Function: llm.ToolCallFunction{Name: tool, Arguments: "{}"}},
		}}, nil
	}
	return llm.Message{Role: llm.RoleAssistant, Content: s.answers[question]}, nil
}

type okTools struct{}

func (okTools) Definitions() []llm.ToolDefinition   { return nil }
func (okTools) Call(string, string) (string, error) { return "result", nil }

func newAgent(m agent.Model) *agent.Agent {
	return &agent.Agent{Model: m, Tools: okTools{}, MaxIterations: 3}
}

func TestRunAllPass(t *testing.T) {
	cases := []Case{
		{Question: "q1", ExpectContains: []string{"a1"}},
		{Question: "q2", ExpectContains: []string{"a2"}, ExpectTool: "list_files"},
	}
	m := &scriptedModel{answers: map[string]string{"q1": "a1", "q2": "a2"}, callTools: map[string]string{"q2": "list_files"}}

	results := Run(context.Background(), newAgent(m), cases, 0, nil)
	s := Summarize(results)
	if s.Passed != 2 || s.Total != 2 || s.Rate() != 100 {
		t.Fatalf("summary = %+v", s)
	}
}

func TestRunSomeFail(t *testing.T) {
	cases := []Case{
		{Question: "q1", ExpectContains: []string{"a1"}},
		{Question: "q2", ExpectContains: []string{"a2"}},
	}
	m := &scriptedModel{answers: map[string]string{"q1": "a1", "q2": "wrong"}}

	var seen []int
	results := Run(context.Background(), newAgent(m), cases, 0, func(i int, _ Result) { seen = append(seen, i) })
	s := Summarize(results)
	if s.Passed != 1 || s.Rate() != 50 {
		t.Fatalf("summary = %+v", s)
	}
	if results[1].Passed || results[1].Answer != "wrong" {
		t.Fatalf("result = %+v", results[1])
	}
	if len(seen) != 2 {
		t.Fatalf("onResult calls = %v", seen)
	}
	out := FormatResult(1, 2, results[1])
	if !strings.Contains(out, "FAIL") || !strings.Contains(out, "q2") || !strings.Contains(out, "answer: wrong") {
		t.Fatalf("format = %q", out)
	}
}

func TestRunErrorFailsCaseAndContinues(t *testing.T) {
	cases := []Case{
		{Question: "q1", ExpectContains: []string{"a1"}},
		{Question: "q2", ExpectContains: []string{"a2"}},
	}
	m := &scriptedModel{answers: map[string]string{"q1": "a1", "q2": "a2"}, failOn: "q1"}

	results := Run(context.Background(), newAgent(m), cases, 0, nil)
	if results[0].Passed || results[0].Err == nil || !strings.Contains(results[0].Reason, "network down") {
		t.Fatalf("first = %+v", results[0])
	}
	if !results[1].Passed {
		t.Fatalf("second case should still run and pass: %+v", results[1])
	}
}

func TestRunUsesFreshHistoryPerCase(t *testing.T) {
	var lens []int
	m := &lenModel{lens: &lens}
	cases := []Case{
		{Question: "q1", ExpectContains: []string{"ok"}},
		{Question: "q2", ExpectContains: []string{"ok"}},
	}
	Run(context.Background(), newAgent(m), cases, time.Millisecond, nil)
	if len(lens) != 2 || lens[0] != lens[1] {
		t.Fatalf("history sizes = %v, want equal", lens)
	}
}

type lenModel struct{ lens *[]int }

func (l *lenModel) Chat(_ context.Context, msgs []llm.Message, _ []llm.ToolDefinition) (llm.Message, error) {
	*l.lens = append(*l.lens, len(msgs))
	return llm.Message{Role: llm.RoleAssistant, Content: "ok"}, nil
}
