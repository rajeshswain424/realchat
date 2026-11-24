# RealChat Roadmap

RealChat is a notes and docs agent. You ask questions about a folder of your own markdown files. An LLM-driven agent searches and reads them with tools, then answers. One project grows through four phases, so every phase builds on the same agent core.

## Stack and decisions

- Backend: Go, with no agent framework. The agent loop, tool dispatch and context handling are written by hand.
- HTTP server (Phase 2+): `chi` router instead of plain `net/http` handlers. The LLM client still uses the standard library `net/http` client, because chi is a server-side router.
- Streaming: server-sent events (SSE).
- Storage: SQLite for traces.
- Frontend (Phase 2+): React.
- LLM API: OpenAI-compatible chat completions. Groq first, then a self-hosted model through Ollama or vLLM.

## Architecture

```
 You: "What did I decide about auth?"
        |
        v
 +---------------------------+
 |        AGENT LOOP         |
 |  messages --> LLM         |
 |     ^          |          |
 |     |     tool call?      |
 |     |       yes | no      |
 |     |        v  +--> final answer
 |  tool result <- run tool  |
 +---------------------------+
        |          |
   list/read/search  |
   (your .md files)  v
                 trace -> SQLite
```

The loop emits events through a callback. The CLI, the SSE stream, the trace store and the eval runner all consume those same events.

## Phases and milestones

```
 M0 --> M1 --> M2 --> M3 --> M4 --> M5      Phase 1: core agent (CLI)
                                 |
 M6 --> M7 --> M8 --> M9 --> M10            Phase 2: harness (API + UI)
                                 |
 M11 --> M12                                Phase 3: inference
                                 |
 M13 --> M14                                Phase 4: audio
```

### Phase 1: CLI agent

| # | Milestone | You learn | Result |
|---|-----------|-----------|--------|
| M0 | Go module and one LLM call | Chat-completions request/response format, API keys, env vars | A program that prints a reply from Groq |
| M1 | Message and tool structs | Roles, tool-call JSON schema | Go types matching the API, multi-turn chat in the terminal |
| M2 | Single tool call | How the model requests a tool and how you return the result | The model calls `list_files` once and uses the output |
| M3 | The agent loop | Loop until no tool calls, iteration cap, tool errors fed back to the model | A working agent with `list_files`, `read_file`, `search_text` |
| M4 | Safety and sandbox | Path traversal, restricting reads to one folder, output truncation | Tools that cannot escape the notes folder |
| M5 | Mini eval | Fixed Q&A set, pass/fail checks, non-determinism | An eval runner that prints a pass rate (baseline score) |

### Phase 2: Harness

| # | Milestone | You learn | Result |
|---|-----------|-----------|--------|
| M6 | HTTP API (chi) | chi routing, middleware, structured logging, refactoring the loop to emit events | `POST /chat` returns the full answer |
| M7 | SSE streaming | `http.Flusher`, event types (`tool_start`, `tool_result`, `token`, `done`) | `curl` shows live progress |
| M8 | React chat UI | Streaming in the browser, rendering tool calls live | A chat page where you watch the agent think |
| M9 | Permission system | A write tool, pausing the loop for human approval | `write_note` waits for an approve or deny click |
| M10 | Traces and compaction | Persisting runs to SQLite, a trace viewer, summarizing old context | A trace page, and long sessions that do not overflow |

### Phase 3: Inference

| # | Milestone | You learn | Result |
|---|-----------|-----------|--------|
| M11 | Swap the backend | OpenAI-compatible APIs, base-URL config | The same agent running on Ollama or vLLM |
| M12 | Benchmark | Accuracy vs latency vs cost, small-model tool-calling failures | A comparison table from the eval runner |

### Phase 4: Audio

| # | Milestone | You learn | Result |
|---|-----------|-----------|--------|
| M13 | Audio in | Microphone capture, WebSocket to Go, speech-to-text | Speech becomes agent input |
| M14 | Audio out | Text-to-speech and streaming it back | A full voice conversation |

## End goal

An agent that is measured (eval pass rate over time), observable (trace viewer) and safe (permission flow), with the model backend swappable and a voice mode on top.

## Working rules

- Stay on one model provider until M5, so bugs are not confused with model changes.
- The agent loop emits events from the start, so later phases reuse it unchanged.
- Each phase is one or more OpenSpec changes under `openspec/changes/`.
