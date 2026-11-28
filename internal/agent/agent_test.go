package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"realchat/internal/llm"
)

// fakeModel returns scripted replies in order and records what it was sent.
type fakeModel struct {
	replies []llm.Message
	err     error
	calls   [][]llm.Message
	tools   [][]llm.ToolDefinition
}

func (f *fakeModel) Chat(_ context.Context, msgs []llm.Message, tools []llm.ToolDefinition) (llm.Message, error) {
	f.calls = append(f.calls, append([]llm.Message(nil), msgs...))
	f.tools = append(f.tools, tools)
	if f.err != nil {
		return llm.Message{}, f.err
	}
	if len(f.replies) == 0 {
		return llm.Message{}, errors.New("fakeModel: no scripted reply left")
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	return r, nil
}

// fakeTools runs named functions and counts invocations.
type fakeTools struct {
	funcs map[string]func(args string) (string, error)
	ran   []string
}

func (f *fakeTools) Definitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{llm.NewToolDefinition("echo", "echo", []byte(`{"type":"object"}`))}
}

func (f *fakeTools) Call(name, args string) (string, error) {
	fn, ok := f.funcs[name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", name)
	}
	f.ran = append(f.ran, name)
	return fn(args)
}

func toolCall(id, name, args string) llm.ToolCall {
	return llm.ToolCall{ID: id, Type: "function", Function: llm.ToolCallFunction{Name: name, Arguments: args}}
}

func assistantTools(calls ...llm.ToolCall) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, ToolCalls: calls}
}

func assistantText(s string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Content: s}
}

func newTools() *fakeTools {
	return &fakeTools{funcs: map[string]func(string) (string, error){
		"echo": func(args string) (string, error) { return "echoed " + args, nil },
		"fail": func(string) (string, error) { return "", errors.New("boom") },
	}}
}

func TestAnswerWithoutTools(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{assistantText("hello")}}
	a := &Agent{Model: m, Tools: newTools()}

	got, hist, err := a.Run(context.Background(), nil, "hi", nil)
	if err != nil || got != "hello" {
		t.Fatalf("got %q, %v", got, err)
	}
	if len(m.calls) != 1 {
		t.Fatalf("model calls = %d, want 1", len(m.calls))
	}
	if len(hist) != 2 || hist[0].Role != llm.RoleUser || hist[1].Role != llm.RoleAssistant {
		t.Fatalf("history = %+v", hist)
	}
	if len(m.tools[0]) != 1 {
		t.Fatal("tool definitions were not sent to the model")
	}
}

func TestOneToolThenAnswer(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{
		assistantTools(toolCall("c1", "echo", `{"x":1}`)),
		assistantText("done"),
	}}
	tl := newTools()
	a := &Agent{Model: m, Tools: tl}

	got, hist, err := a.Run(context.Background(), nil, "go", nil)
	if err != nil || got != "done" {
		t.Fatalf("got %q, %v", got, err)
	}
	if len(tl.ran) != 1 {
		t.Fatalf("tool ran %d times", len(tl.ran))
	}

	second := m.calls[1]
	last := second[len(second)-1]
	if last.Role != llm.RoleTool || last.ToolCallID != "c1" || last.Content != `echoed {"x":1}` {
		t.Fatalf("tool message sent to model = %+v", last)
	}
	// user, assistant{tool_calls}, tool, assistant
	if len(hist) != 4 || len(hist[1].ToolCalls) != 1 {
		t.Fatalf("history = %+v", hist)
	}
}

func TestTwoToolCallsInOneReply(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{
		assistantTools(toolCall("a", "echo", `1`), toolCall("b", "echo", `2`)),
		assistantText("both done"),
	}}
	a := &Agent{Model: m, Tools: newTools()}

	if _, _, err := a.Run(context.Background(), nil, "go", nil); err != nil {
		t.Fatal(err)
	}
	second := m.calls[1]
	n := len(second)
	if second[n-2].ToolCallID != "a" || second[n-1].ToolCallID != "b" {
		t.Fatalf("tool results out of order: %+v", second[n-2:])
	}
}

func TestIterationCap(t *testing.T) {
	loop := assistantTools(toolCall("c", "echo", `{}`))
	m := &fakeModel{replies: []llm.Message{loop, loop, loop, loop, loop}}
	a := &Agent{Model: m, Tools: newTools(), MaxIterations: 3}

	var events []Event
	_, hist, err := a.Run(context.Background(), nil, "go", func(e Event) { events = append(events, e) })
	if !errors.Is(err, ErrIterationLimit) {
		t.Fatalf("want ErrIterationLimit, got %v", err)
	}
	if len(m.calls) != 3 {
		t.Fatalf("model calls = %d, want 3", len(m.calls))
	}
	if events[len(events)-1].Kind != EventLimitReached {
		t.Fatalf("last event = %+v", events[len(events)-1])
	}
	if len(hist) < 3 {
		t.Fatal("partial history should be returned")
	}
}

func TestToolErrorIsSentBackAndRunContinues(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{
		assistantTools(toolCall("c1", "fail", `{}`)),
		assistantText("recovered"),
	}}
	a := &Agent{Model: m, Tools: newTools()}

	var results []Event
	got, _, err := a.Run(context.Background(), nil, "go", func(e Event) {
		if e.Kind == EventToolResult {
			results = append(results, e)
		}
	})
	if err != nil || got != "recovered" {
		t.Fatalf("got %q, %v", got, err)
	}
	second := m.calls[1]
	if msg := second[len(second)-1]; !strings.Contains(msg.Content, "boom") {
		t.Fatalf("error text not sent to model: %+v", msg)
	}
	if len(results) != 1 || !results[0].IsError {
		t.Fatalf("result events = %+v", results)
	}
}

func TestUnknownToolIsSentBack(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{
		assistantTools(toolCall("c1", "nope", `{}`)),
		assistantText("ok"),
	}}
	a := &Agent{Model: m, Tools: newTools()}

	if _, _, err := a.Run(context.Background(), nil, "go", nil); err != nil {
		t.Fatal(err)
	}
	second := m.calls[1]
	if msg := second[len(second)-1]; !strings.Contains(msg.Content, "unknown tool") {
		t.Fatalf("got %+v", msg)
	}
}

func TestInvalidArgumentsAreSentBack(t *testing.T) {
	tl := newTools()
	tl.funcs["strict"] = func(args string) (string, error) { return "", fmt.Errorf("invalid arguments: %s", args) }
	m := &fakeModel{replies: []llm.Message{
		assistantTools(toolCall("c1", "strict", `{bad`)),
		assistantText("ok"),
	}}
	a := &Agent{Model: m, Tools: tl}

	if _, _, err := a.Run(context.Background(), nil, "go", nil); err != nil {
		t.Fatal(err)
	}
	second := m.calls[1]
	if msg := second[len(second)-1]; !strings.Contains(msg.Content, "invalid arguments") {
		t.Fatalf("got %+v", msg)
	}
}

func TestEventOrder(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{
		assistantTools(toolCall("c1", "echo", `{}`)),
		assistantText("fin"),
	}}
	a := &Agent{Model: m, Tools: newTools()}

	var kinds []EventKind
	if _, _, err := a.Run(context.Background(), nil, "go", func(e Event) { kinds = append(kinds, e.Kind) }); err != nil {
		t.Fatal(err)
	}
	want := []EventKind{EventModelCall, EventToolStart, EventToolResult, EventModelCall, EventFinal}
	if fmt.Sprint(kinds) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", kinds, want)
	}
}

func TestNilHandlerIsAllowed(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{
		assistantTools(toolCall("c1", "echo", `{}`)),
		assistantText("fin"),
	}}
	a := &Agent{Model: m, Tools: newTools()}

	if got, _, err := a.Run(context.Background(), nil, "go", nil); err != nil || got != "fin" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestModelErrorEndsRun(t *testing.T) {
	boom := errors.New("network down")
	a := &Agent{Model: &fakeModel{err: boom}, Tools: newTools()}

	_, hist, err := a.Run(context.Background(), nil, "go", nil)
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	if len(hist) != 1 || hist[0].Role != llm.RoleUser {
		t.Fatalf("history = %+v", hist)
	}
}

func TestRunDoesNotMutateInputHistory(t *testing.T) {
	m := &fakeModel{replies: []llm.Message{assistantText("a")}}
	a := &Agent{Model: m, Tools: newTools()}

	in := make([]llm.Message, 1, 10) // spare capacity would expose aliasing bugs
	in[0] = llm.Message{Role: llm.RoleSystem, Content: "sys"}
	_, out, err := a.Run(context.Background(), in, "q", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 1 || len(out) != 3 {
		t.Fatalf("in=%d out=%d", len(in), len(out))
	}
	if in[:2][1].Content != "" {
		t.Fatal("input history backing array was written to")
	}
}
