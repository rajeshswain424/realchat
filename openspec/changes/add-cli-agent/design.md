# Design

## Context

The repository is empty, with Go 1.22.4 installed and no third-party dependencies. See proposal.md for motivation. Phase 1 must produce an agent core that Phase 2 can wrap in a `chi` HTTP server and an SSE stream without rewriting it. The model provider is Groq, which exposes an OpenAI-compatible chat-completions API.

## Goals / Non-Goals

**Goals:**
- A small, readable agent core written by hand, where each piece (client, loop, tools, eval) can be understood on its own.
- A core that is independent of the terminal, so a CLI, an HTTP handler and an eval runner can all drive it.
- Tests that run without network access.

**Non-Goals:**
- Streaming token output from the model. Phase 1 uses non-streamed responses.
- Parallel tool execution. Tool calls run sequentially.
- Any HTTP server code. `chi` is added in Phase 2.

## Decisions

### Layout: one module, `cmd/` plus `internal/`

```
 realchat/
   go.mod                      module realchat
   cmd/agent/main.go           CLI
   cmd/eval/main.go            eval runner
   internal/llm/               client, message and tool types
   internal/agent/             loop, events
   internal/tools/             list_files, read_file, search_text, registry
   internal/eval/              case loading and checking
   testdata/notes/             sample corpus
   testdata/eval.json          eval cases
```

Dependencies point inward: `cmd` -> `agent` -> `llm` and `tools`. `llm` and `tools` do not import `agent`. `internal/` keeps the packages private until there is a reason to export them. Alternative: a single `main` package. Rejected because it makes Phase 2 reuse harder.

### Standard library only, including the LLM call
The LLM client uses `net/http` and `encoding/json` directly, with a client-level timeout. Writing the request and response structs by hand is the learning goal. Alternative: the official OpenAI Go SDK. Rejected for now because it hides the wire format, and a thin hand-written client is under 150 lines. `chi` is a server router, so it does not apply to this outbound client. It is added with the HTTP server in Phase 2.

### Model access behind a small interface
`agent` depends on an interface with a single `Chat(ctx, messages, tools)` method, which `llm.Client` implements. Tests use a scripted fake that returns canned replies, so loop behavior (tool cycle, cap, error recovery) is tested without network access. The interface also makes the Phase 3 backend swap a configuration change only.

### Configuration through environment variables
`GROQ_API_KEY`, `LLM_BASE_URL` (default `https://api.groq.com/openai/v1`, with `/chat/completions` appended by the client) and `LLM_MODEL` (default `openai/gpt-oss-120b`). Using a generic base-URL name now means Phase 3 needs no renames. The key is sent as a `Bearer` token and is never logged or included in errors. Alternative: a config file. Rejected as unnecessary for one user.

### Request parameters follow Groq's documented body
Requests send `model`, `messages`, `tools`, `temperature`, `max_completion_tokens`, `reasoning_effort` and `stream: false`. Groq uses `max_completion_tokens` rather than `max_tokens`. Temperature, token limit and reasoning effort have defaults (temperature 0 for evals, 2048 tokens, `medium`) and are overridable by flags or environment variables. Unset optional fields are omitted from the JSON.

### Messages and tool calls mirror the wire format
Types carry `role`, `content`, `tool_calls` and `tool_call_id`. Tool arguments are kept as a raw JSON string, because that is what the API sends, and each tool decodes its own arguments. Assistant messages with tool calls are appended to history exactly as received, which the API requires before the tool results.

### Loop emits events through a callback
`Run(ctx, history, userInput, onEvent)` returns the final answer and the updated history. Event kinds are model call, tool start, tool result, final answer and limit reached. The CLI prints events, the eval runner records which tools were called, and Phase 2 forwards them to SSE. The loop never prints. A nil handler is allowed.

### Errors are results, not exceptions
A tool error, unknown tool or bad arguments becomes a tool-role message with the error text, so the model can correct itself. Only transport-level failures from the LLM client end the run. The iteration cap (default 10) is checked before each model call and ends the run with a distinct limit error.

### Sandbox tools by resolving paths against the notes root
Every path argument is cleaned, joined to the root, resolved through symlinks, and rejected unless the result stays inside the resolved root. `search_text` and `list_files` skip entries that resolve outside the root. Alternative: `os.Root` (Go 1.24). Rejected because 1.22 is installed, so the check is hand-written and covered by traversal tests. Output limits are fixed byte and match counts, with an explicit "truncated" note so the model knows it saw part of the data.

### Eval: substring checks over a bundled corpus
Cases are JSON: `question`, `expect_contains` (list, all must appear, case-insensitive) and optional `expect_tool`. The corpus lives in `testdata/notes` so results are reproducible for anyone. Substring checks are crude but deterministic and free. An LLM judge is deferred, because it adds a second model whose own accuracy would need checking. The runner exits non-zero on any failure and prints the pass rate, so the baseline can be recorded.

## Risks / Trade-offs

- [Model output is non-deterministic, so eval scores vary between runs] -> Use temperature 0 and keep checks lenient on wording. Record the model name next to each score.
- [A reasoning model may return a `reasoning` field or use part of the token budget on reasoning] -> Ignore unknown response fields, and keep the token limit high enough for reasoning plus the answer.
- [Not every Groq model supports tool calling or `reasoning_effort`] -> Default to `openai/gpt-oss-120b`, and surface the provider's error message when a model rejects a parameter.
- [Groq free-tier rate limits can fail eval cases] -> Treat errors as failed cases with the reason shown. Add a short delay between cases if limits are hit.
- [Hand-written path sandboxing can have gaps, especially on Windows] -> Test traversal, absolute paths, drive-letter paths and symlinks. Symlink creation may need privileges on Windows, so those tests skip when it cannot be created.
- [The hand-written client only covers the fields used] -> Acceptable for Phase 1. Extend it when a later phase needs more.
- [Entire-file reads can overflow the context window] -> Fixed output cap now. Real compaction comes in Phase 2, M10.
