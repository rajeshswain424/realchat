# Spec Delta

## Purpose

Defines the cycle in which the agent asks the model, runs the tools it requests, feeds results back, and finishes with an answer, with bounded iterations and observable progress.

## ADDED Requirements

### Requirement: Answer without tools
The agent SHALL return the model's reply as the final answer when the reply contains no tool calls.

#### Scenario: Direct answer
- **WHEN** the model replies with text and no tool calls
- **THEN** the agent returns that text as the final answer after one model call

### Requirement: Tool execution cycle
The agent SHALL run every tool call the model requests, append each result to the conversation with the matching tool call id, and call the model again until it replies without tool calls.

#### Scenario: One tool then answer
- **WHEN** the model requests `list_files` and then answers using the result
- **THEN** the agent runs the tool once, sends the result back, and returns the second reply as the final answer

#### Scenario: Several tool calls in one reply
- **WHEN** the model requests two tool calls in one reply
- **THEN** the agent runs both and sends one result per call id before the next model call

### Requirement: Iteration cap
The agent SHALL stop after a configurable maximum number of model calls and SHALL report that the cap was reached instead of looping forever.

#### Scenario: Model never stops calling tools
- **WHEN** the model keeps requesting tools past the cap
- **THEN** the agent stops, reports that the iteration limit was reached, and returns any partial progress

### Requirement: Tool failures do not abort the run
The agent SHALL send a tool's error message back to the model as that tool's result and SHALL continue, including when the model names an unknown tool or sends invalid arguments.

#### Scenario: Tool returns an error
- **WHEN** a tool fails with an error
- **THEN** the error text is sent to the model as the tool result and the loop continues

#### Scenario: Unknown tool name
- **WHEN** the model requests a tool that does not exist
- **THEN** the agent sends back an error result that says the tool is unknown and continues

#### Scenario: Invalid arguments
- **WHEN** the tool arguments are not valid JSON for that tool
- **THEN** the agent sends back an error result describing the problem and continues

### Requirement: Progress events
The agent SHALL report progress through a caller-supplied event handler, with events for each model call, each tool call start, each tool result, and the final answer, and SHALL NOT write to the terminal itself.

#### Scenario: Events in order
- **WHEN** a run includes one tool call
- **THEN** the handler receives the tool call start, the tool result, and the final answer, in that order

#### Scenario: No handler supplied
- **WHEN** the agent runs without an event handler
- **THEN** it completes normally and produces no output of its own
