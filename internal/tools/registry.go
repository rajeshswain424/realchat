package tools

import (
	"encoding/json"
	"fmt"

	"realchat/internal/llm"
)

// Tool is one capability the model can call.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage // JSON schema for the arguments
	run         func(args json.RawMessage) (string, error)
}

// Registry holds tools in a fixed order and dispatches calls by name.
type Registry struct {
	tools  []Tool
	byName map[string]Tool
}

// NewNotesRegistry returns the three read-only notes tools bound to root.
func NewNotesRegistry(root *Root) *Registry {
	return newRegistry(
		Tool{
			Name:        "list_files",
			Description: "List all markdown notes in the notes folder as relative paths.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
			run:         root.listFiles,
		},
		Tool{
			Name:        "read_file",
			Description: "Read the full text of one markdown note by its relative path, for example projects/realchat.md.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Relative path of the note, as returned by list_files."}},"required":["path"],"additionalProperties":false}`),
			run:         root.readFile,
		},
		Tool{
			Name:        "search_text",
			Description: "Search all notes for a text snippet, case-insensitively. Returns matching lines as path:line: text.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Text to look for."}},"required":["query"],"additionalProperties":false}`),
			run:         root.searchText,
		},
	)
}

func newRegistry(tools ...Tool) *Registry {
	r := &Registry{tools: tools, byName: make(map[string]Tool, len(tools))}
	for _, t := range tools {
		r.byName[t.Name] = t
	}
	return r
}

// Definitions returns the tool definitions to send to the model.
func (r *Registry) Definitions() []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, llm.NewToolDefinition(t.Name, t.Description, t.Parameters))
	}
	return defs
}

// Call runs the named tool with raw JSON arguments.
func (r *Registry) Call(name, arguments string) (string, error) {
	t, ok := r.byName[name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", name)
	}
	if arguments == "" {
		arguments = "{}"
	}
	if !json.Valid([]byte(arguments)) {
		return "", fmt.Errorf("invalid arguments for %s: not valid JSON", name)
	}
	return t.run(json.RawMessage(arguments))
}
