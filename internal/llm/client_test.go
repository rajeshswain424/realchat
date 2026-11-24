package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testConfig(url string) Config {
	temp := 0.0
	return Config{
		APIKey:              "secret-key",
		BaseURL:             url,
		Model:               "test-model",
		Temperature:         &temp,
		MaxCompletionTokens: 100,
		Timeout:             2 * time.Second,
	}
}

func TestChatSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-key" {
			t.Errorf("auth header = %q", got)
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
	defer srv.Close()

	reply, err := NewClient(testConfig(srv.URL)).Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Role != RoleAssistant || reply.Content != "hello" {
		t.Fatalf("reply = %+v", reply)
	}
}

func TestChatSendsHistoryAndToolsAndDecodesToolCalls(t *testing.T) {
	var got struct {
		Messages []Message        `json:"messages"`
		Tools    []ToolDefinition `json:"tools"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_9","type":"function","function":{"name":"list_files","arguments":"{}"}}]}}]}`))
	}))
	defer srv.Close()

	history := []Message{
		{Role: RoleSystem, Content: "be brief"},
		{Role: RoleUser, Content: "what notes exist?"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "call_1", Type: "function", Function: ToolCallFunction{Name: "list_files", Arguments: "{}"}}}},
		{Role: RoleTool, ToolCallID: "call_1", Content: "a.md"},
	}
	tools := []ToolDefinition{NewToolDefinition("list_files", "List notes", json.RawMessage(`{"type":"object","properties":{}}`))}

	reply, err := NewClient(testConfig(srv.URL)).Chat(context.Background(), history, tools)
	if err != nil {
		t.Fatal(err)
	}

	wantRoles := []string{RoleSystem, RoleUser, RoleAssistant, RoleTool}
	if len(got.Messages) != len(wantRoles) {
		t.Fatalf("sent %d messages", len(got.Messages))
	}
	for i, role := range wantRoles {
		if got.Messages[i].Role != role {
			t.Errorf("message %d role = %q, want %q", i, got.Messages[i].Role, role)
		}
	}
	if got.Messages[3].ToolCallID != "call_1" {
		t.Errorf("tool message id = %q", got.Messages[3].ToolCallID)
	}
	if len(got.Tools) != 1 || got.Tools[0].Function.Name != "list_files" {
		t.Errorf("tools = %+v", got.Tools)
	}

	if len(reply.ToolCalls) != 1 || reply.ToolCalls[0].ID != "call_9" || reply.ToolCalls[0].Function.Name != "list_files" {
		t.Fatalf("reply tool calls = %+v", reply.ToolCalls)
	}
}

func TestChatAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	_, err := NewClient(testConfig(srv.URL)).Chat(context.Background(), nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %v", err)
	}
	if apiErr.Status != 401 || apiErr.Message != "invalid api key" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatal("error leaks the API key")
	}
}

func TestChatTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.Timeout = 50 * time.Millisecond

	_, err := NewClient(cfg).Chat(context.Background(), nil, nil)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
}
