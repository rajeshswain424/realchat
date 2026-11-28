package agent

import (
	"context"
	"errors"
	"fmt"

	"realchat/internal/llm"
)

// DefaultMaxIterations caps the number of model calls in a single run.
const DefaultMaxIterations = 10

// SystemPrompt tells the model what it is for and how to use the notes tools.
const SystemPrompt = `You answer questions about the user's markdown notes.
Use the tools to find the information: search or list the notes first, then read the relevant files.
Answer only from what the notes say. If the notes do not contain the answer, say so.
Keep answers short and mention which note the answer came from.`

// ErrIterationLimit is returned when the model is still calling tools after the cap.
var ErrIterationLimit = errors.New("iteration limit reached")

// Model is the part of the LLM client the agent needs.
type Model interface {
	Chat(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (llm.Message, error)
}

// Tools is the part of the tool registry the agent needs.
type Tools interface {
	Definitions() []llm.ToolDefinition
	Call(name, arguments string) (string, error)
}

// EventKind identifies what happened in a run.
type EventKind string

const (
	EventModelCall    EventKind = "model_call"
	EventToolStart    EventKind = "tool_start"
	EventToolResult   EventKind = "tool_result"
	EventFinal        EventKind = "final"
	EventLimitReached EventKind = "limit_reached"
)

// Event reports progress. Only the fields relevant to Kind are set.
type Event struct {
	Kind       EventKind
	Iteration  int    // 1-based model call number
	ToolCallID string // tool_start, tool_result
	ToolName   string // tool_start, tool_result
	ToolArgs   string // tool_start
	Result     string // tool_result: the output, or the error text when IsError
	IsError    bool   // tool_result
	Text       string // final
}

// Agent runs the model-and-tools cycle.
type Agent struct {
	Model         Model
	Tools         Tools
	MaxIterations int
}

// Run appends userInput to history and loops until the model answers without tool calls.
// It returns the final answer and the updated history, and never mutates the history passed in.
// onEvent may be nil. On error the returned history still holds everything exchanged so far.
func (a *Agent) Run(ctx context.Context, history []llm.Message, userInput string, onEvent func(Event)) (string, []llm.Message, error) {
	emit := func(e Event) {
		if onEvent != nil {
			onEvent(e)
		}
	}
	limit := a.MaxIterations
	if limit <= 0 {
		limit = DefaultMaxIterations
	}

	msgs := make([]llm.Message, len(history), len(history)+2)
	copy(msgs, history)
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: userInput})

	defs := a.Tools.Definitions()
	for i := 1; ; i++ {
		if i > limit {
			emit(Event{Kind: EventLimitReached, Iteration: limit})
			return "", msgs, fmt.Errorf("%w (%d model calls)", ErrIterationLimit, limit)
		}

		emit(Event{Kind: EventModelCall, Iteration: i})
		reply, err := a.Model.Chat(ctx, msgs, defs)
		if err != nil {
			return "", msgs, err
		}
		msgs = append(msgs, reply)

		if len(reply.ToolCalls) == 0 {
			emit(Event{Kind: EventFinal, Iteration: i, Text: reply.Content})
			return reply.Content, msgs, nil
		}

		for _, call := range reply.ToolCalls {
			emit(Event{Kind: EventToolStart, Iteration: i, ToolCallID: call.ID, ToolName: call.Function.Name, ToolArgs: call.Function.Arguments})

			// Errors go back to the model as the tool result so it can correct itself.
			result, err := a.Tools.Call(call.Function.Name, call.Function.Arguments)
			isErr := err != nil
			if isErr {
				result = "error: " + err.Error()
			}

			emit(Event{Kind: EventToolResult, Iteration: i, ToolCallID: call.ID, ToolName: call.Function.Name, Result: result, IsError: isErr})
			msgs = append(msgs, llm.Message{Role: llm.RoleTool, ToolCallID: call.ID, Content: result})
		}
	}
}
