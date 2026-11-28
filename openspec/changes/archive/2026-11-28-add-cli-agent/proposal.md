# Proposal

## Why

RealChat is an empty repository. The whole roadmap (HTTP API, UI, traces, self-hosted models, voice) depends on a working agent core, so that core has to exist first. Building it as a terminal program keeps the learning focus on the agent loop, tool calling and evaluation, with no UI or server to distract from it. A baseline eval score is also needed now, so later changes can be compared against it.

## What Changes

- Create the Go module and project layout.
- Add an LLM client for an OpenAI-compatible chat-completions endpoint (Groq first), configured by environment variables, using the standard library HTTP client.
- Add typed messages and tool-call structures that match the API.
- Add the agent loop: call the model, run requested tools, feed results back, stop on a final answer or at an iteration cap. The loop reports progress through an event callback instead of printing directly.
- Add three read-only tools over a notes folder: `list_files`, `read_file`, `search_text`. They are confined to that folder and cap their output size.
- Add a CLI that takes a question, or runs an interactive session, and prints tool activity and the final answer.
- Add a basic eval runner: a fixed question set over a sample notes folder, with pass/fail checks and a printed pass rate.
- Out of scope: HTTP server, `chi` routing, SSE, UI, write tools, SQLite traces, context compaction, other model hosts, audio. These belong to later phases in [ROADMAP.md](../../../ROADMAP.md).

## Capabilities

### New Capabilities
- `llm-client`: Sending chat-completion requests, with tool definitions, to an OpenAI-compatible endpoint and decoding replies and errors.
- `agent-loop`: Running the model-and-tool cycle with an iteration cap, tool-error recovery and progress events.
- `notes-tools`: The read-only `list_files`, `read_file` and `search_text` tools, their schemas, and their confinement to the notes folder.
- `agent-cli`: The terminal entry point for asking questions and showing agent progress.
- `agent-eval`: Running a fixed question set against the agent and reporting a pass rate.

### Modified Capabilities

## Impact

- New Go module with `cmd/`, `internal/` packages and a sample notes folder for evals. No existing code is affected.
- External dependency: a Groq API key supplied through an environment variable. It is never stored in the repo.
- Toolchain: Go 1.22.4 is installed. The Phase 1 code has no third-party dependencies. `chi` is introduced in Phase 2.
- Network access to the LLM provider is required to run the agent and the evals.
