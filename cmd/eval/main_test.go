package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"realchat/internal/agent"
	"realchat/internal/llm"
)

const notesPath = "../../testdata/notes"

// answerModel replies with a fixed answer per question and fails for the question named in failOn.
type answerModel struct {
	answers map[string]string
	failOn  string
}

func (a *answerModel) Chat(_ context.Context, msgs []llm.Message, _ []llm.ToolDefinition) (llm.Message, error) {
	var q string
	for _, m := range msgs {
		if m.Role == llm.RoleUser {
			q = m.Content
		}
	}
	if q == a.failOn {
		return llm.Message{}, errors.New("network down")
	}
	return llm.Message{Role: llm.RoleAssistant, Content: a.answers[q]}, nil
}

func setup(t *testing.T, m *answerModel, cases string) string {
	t.Helper()
	t.Setenv(llm.EnvAPIKey, "k")
	old := newModel
	newModel = func(llm.Config) agent.Model { return m }
	t.Cleanup(func() { newModel = old })

	p := filepath.Join(t.TempDir(), "eval.json")
	if err := os.WriteFile(p, []byte(cases), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const twoCases = `[
  {"question":"q1","expect_contains":["a1"]},
  {"question":"q2","expect_contains":["a2"]}
]`

func runEval(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestAllPassExitsZero(t *testing.T) {
	p := setup(t, &answerModel{answers: map[string]string{"q1": "a1", "q2": "a2"}}, twoCases)

	code, out, stderr := runEval("-cases", p, "-notes", notesPath)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(out, "Passed: 2/2 (100.0%)") || strings.Contains(out, "FAIL") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestSomeFailExitsNonZero(t *testing.T) {
	p := setup(t, &answerModel{answers: map[string]string{"q1": "a1", "q2": "wrong"}}, twoCases)

	code, out, _ := runEval("-cases", p, "-notes", notesPath)
	if code != 1 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "Passed: 1/2 (50.0%)") || !strings.Contains(out, "FAIL [2/2] q2") || !strings.Contains(out, "answer: wrong") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestErrorCaseFailsAndRunContinues(t *testing.T) {
	p := setup(t, &answerModel{answers: map[string]string{"q1": "a1", "q2": "a2"}, failOn: "q1"}, twoCases)

	code, out, _ := runEval("-cases", p, "-notes", notesPath)
	if code != 1 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "network down") || !strings.Contains(out, "PASS [2/2] q2") || !strings.Contains(out, "Passed: 1/2") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestMalformedCasesFile(t *testing.T) {
	p := setup(t, &answerModel{}, `[{"question":"q"}]`)

	code, _, stderr := runEval("-cases", p, "-notes", notesPath)
	if code != 1 || !strings.Contains(stderr, "expect_contains") {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
}

func TestMissingAPIKey(t *testing.T) {
	p := setup(t, &answerModel{}, twoCases)
	t.Setenv(llm.EnvAPIKey, "")

	code, _, stderr := runEval("-cases", p, "-notes", notesPath)
	if code != 1 || !strings.Contains(stderr, llm.EnvAPIKey) {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
}

func TestBundledSampleNotesAndCasesLoad(t *testing.T) {
	// Every bundled case is answered wrongly, so this only proves the bundled files wire up and report failures.
	setup(t, &answerModel{}, twoCases)

	code, out, _ := runEval("-cases", "../../testdata/eval.json", "-notes", notesPath)
	if code != 1 || !strings.Contains(out, "Passed: 0/14") {
		t.Fatalf("exit = %d\n%s", code, out)
	}
}
