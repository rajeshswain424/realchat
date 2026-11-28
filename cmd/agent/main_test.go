package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"realchat/internal/agent"
	"realchat/internal/llm"
)

const notesPath = "../../testdata/notes"

type scriptedModel struct {
	replies []llm.Message
	repeat  *llm.Message // returned forever once replies run out
	calls   [][]llm.Message
	model   string
}

func (s *scriptedModel) Chat(_ context.Context, msgs []llm.Message, _ []llm.ToolDefinition) (llm.Message, error) {
	s.calls = append(s.calls, append([]llm.Message(nil), msgs...))
	if len(s.replies) > 0 {
		r := s.replies[0]
		s.replies = s.replies[1:]
		return r, nil
	}
	if s.repeat != nil {
		return *s.repeat, nil
	}
	return llm.Message{}, errors.New("no scripted reply left")
}

func useModel(t *testing.T, m *scriptedModel) {
	t.Helper()
	old := newModel
	newModel = func(cfg llm.Config) agent.Model {
		m.model = cfg.Model
		return m
	}
	t.Cleanup(func() { newModel = old })
}

func text(s string) llm.Message { return llm.Message{Role: llm.RoleAssistant, Content: s} }

func toolMsg(id, name, args string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{
		{ID: id, Type: "function", Function: llm.ToolCallFunction{Name: name, Arguments: args}},
	}}
}

func runCLI(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestOneShotPrintsAnswerAndToolActivity(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	m := &scriptedModel{replies: []llm.Message{
		toolMsg("c1", "search_text", `{"query":"rate limit"}`),
		text("100 requests per minute."),
	}}
	useModel(t, m)

	code, stdout, stderr := runCLI(t, "", "-notes", notesPath, "what is the rate limit?")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "100 requests per minute." {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr, "[tool] search_text") || !strings.Contains(stderr, "decisions.md") {
		t.Fatalf("tool activity missing from stderr: %q", stderr)
	}
}

func TestInteractiveKeepsHistoryAndExitsOnEOF(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	m := &scriptedModel{replies: []llm.Message{text("first"), text("second")}}
	useModel(t, m)

	code, stdout, _ := runCLI(t, "one\ntwo\n", "-notes", notesPath)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "first") || !strings.Contains(stdout, "second") {
		t.Fatalf("stdout = %q", stdout)
	}
	// system, user one, assistant first, user two
	second := m.calls[1]
	if len(second) != 4 || second[1].Content != "one" || second[2].Content != "first" || second[3].Content != "two" {
		t.Fatalf("second call did not include earlier turn: %+v", second)
	}
}

func TestInteractiveExitCommand(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	useModel(t, &scriptedModel{})

	if code, _, _ := runCLI(t, "exit\n", "-notes", notesPath); code != 0 {
		t.Fatalf("exit = %d", code)
	}
}

func TestMissingNotesFolder(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	useModel(t, &scriptedModel{})

	code, _, stderr := runCLI(t, "", "-notes", "no-such-folder", "hi")
	if code == 0 || !strings.Contains(stderr, "no-such-folder") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
}

func TestMissingAPIKey(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "")
	useModel(t, &scriptedModel{})

	code, _, stderr := runCLI(t, "", "-notes", notesPath, "hi")
	if code == 0 || !strings.Contains(stderr, llm.EnvAPIKey) {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
}

func TestIterationLimitExitsNonZero(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	loop := toolMsg("c", "list_files", `{}`)
	useModel(t, &scriptedModel{repeat: &loop})

	code, stdout, stderr := runCLI(t, "", "-notes", notesPath, "-max-iterations", "1", "hi")
	if code == 0 || stdout != "" || !strings.Contains(stderr, "iteration limit") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
}

func TestInteractiveErrorKeepsSessionAlive(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	loop := toolMsg("c", "list_files", `{}`)
	useModel(t, &scriptedModel{repeat: &loop})

	code, _, stderr := runCLI(t, "hi\n", "-notes", notesPath, "-max-iterations", "1")
	if code != 0 || !strings.Contains(stderr, "iteration limit") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
}

func TestModelFlagOverridesEnv(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	t.Setenv(llm.EnvModel, "from-env")
	m := &scriptedModel{replies: []llm.Message{text("ok")}}
	useModel(t, m)

	if code, _, stderr := runCLI(t, "", "-notes", notesPath, "-model", "from-flag", "hi"); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr)
	}
	if m.model != "from-flag" {
		t.Fatalf("model = %q", m.model)
	}
}

func TestInvalidMaxIterations(t *testing.T) {
	t.Setenv(llm.EnvAPIKey, "k")
	useModel(t, &scriptedModel{})

	if code, _, _ := runCLI(t, "", "-notes", notesPath, "-max-iterations", "0", "hi"); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}
