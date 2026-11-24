package llm

import "encoding/json"

// Message is one entry in a conversation, in the provider's wire format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// ToolCalls is set on assistant messages that ask for tools to run.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID is set on tool messages and names the call being answered.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall is a tool invocation requested by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction holds the tool name and its arguments as a raw JSON string, as the API sends it.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition describes a tool the model may call.
type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

// ToolFunctionDefinition carries the name, description and JSON schema of a tool's arguments.
type ToolFunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// NewToolDefinition builds a function-type tool definition.
func NewToolDefinition(name, description string, parameters json.RawMessage) ToolDefinition {
	return ToolDefinition{
		Type:     "function",
		Function: ToolFunctionDefinition{Name: name, Description: description, Parameters: parameters},
	}
}

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)
