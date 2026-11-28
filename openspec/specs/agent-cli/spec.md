# agent-cli Specification

## Purpose

Defines the terminal program through which a user asks the agent questions about a notes folder and watches what the agent does.

## Requirements

### Requirement: One-shot question
The CLI SHALL accept a question as a command-line argument, run the agent once, print the final answer to standard output, and exit.

#### Scenario: Question given
- **WHEN** the user runs the CLI with a question
- **THEN** the final answer is printed and the process exits with status 0

### Requirement: Interactive session
The CLI SHALL start an interactive session when no question is given, keep the conversation history between turns, and end on an exit command or end of input.

#### Scenario: Follow-up question
- **WHEN** the user asks a second question that refers to the first answer
- **THEN** the agent receives the earlier turns as context

#### Scenario: End of input
- **WHEN** the user ends the input stream
- **THEN** the CLI exits with status 0

### Requirement: Visible tool activity
The CLI SHALL print each tool call and a short summary of its result as they happen, separately from the final answer.

#### Scenario: Agent reads a note
- **WHEN** the agent calls `read_file` on `plan.md`
- **THEN** the terminal shows the tool name and arguments before the final answer

### Requirement: Notes folder selection
The CLI SHALL take the notes folder from a flag and SHALL fail with a clear message when the folder does not exist.

#### Scenario: Folder missing
- **WHEN** the given notes folder does not exist
- **THEN** the CLI exits with a non-zero status and a message naming the folder

### Requirement: Failures are reported
The CLI SHALL print configuration, network and iteration-limit failures to standard error and exit with a non-zero status in one-shot mode.

#### Scenario: Missing API key
- **WHEN** the API key variable is not set
- **THEN** the CLI prints which variable is missing and exits with a non-zero status

#### Scenario: Iteration limit reached
- **WHEN** the agent stops at its iteration cap
- **THEN** the CLI prints that the limit was reached and exits with a non-zero status
