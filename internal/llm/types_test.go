package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

// Shape taken from the OpenAI-compatible chat-completions response.
const sampleToolCallReply = `{
  "role": "assistant",
  "content": null,
  "reasoning": "need to list files",
  "tool_calls": [
    {"id": "call_1", "type": "function", "function": {"name": "list_files", "arguments": "{}"}},
    {"id": "call_2", "type": "function", "function": {"name": "read_file", "arguments": "{\"path\":\"a.md\"}"}}
  ]
}`

func TestMessageDecodesToolCalls(t *testing.T) {
	var m Message
	if err := json.Unmarshal([]byte(sampleToolCallReply), &m); err != nil {
		t.Fatal(err)
	}
	if m.Role != RoleAssistant || m.Content != "" {
		t.Fatalf("message = %+v", m)
	}
	if len(m.ToolCalls) != 2 {
		t.Fatalf("tool calls = %d", len(m.ToolCalls))
	}
	if got := m.ToolCalls[1]; got.ID != "call_2" || got.Function.Name != "read_file" || got.Function.Arguments != `{"path":"a.md"}` {
		t.Fatalf("second call = %+v", got)
	}
}

func TestMessageRoundTrip(t *testing.T) {
	var m Message
	if err := json.Unmarshal([]byte(sampleToolCallReply), &m); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var again Message
	if err := json.Unmarshal(data, &again); err != nil {
		t.Fatal(err)
	}
	if len(again.ToolCalls) != 2 || again.ToolCalls[0].ID != "call_1" {
		t.Fatalf("round trip lost tool calls: %s", data)
	}
}

func TestToolMessageEncoding(t *testing.T) {
	data, err := json.Marshal(Message{Role: RoleTool, ToolCallID: "call_1", Content: "a.md"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"role":"tool"`) || !strings.Contains(s, `"tool_call_id":"call_1"`) {
		t.Fatalf("tool message = %s", s)
	}
	if strings.Contains(s, "tool_calls") {
		t.Fatalf("tool_calls should be omitted when empty: %s", s)
	}
}

func TestToolDefinitionEncoding(t *testing.T) {
	def := NewToolDefinition("read_file", "Read a note", json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`))
	data, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"function","function":{"name":"read_file","description":"Read a note","parameters":{"type":"object","properties":{"path":{"type":"string"}}}}}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
}
