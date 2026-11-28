# Tasks

## 1. M0: Project setup and first LLM call

- [x] 1.1 Run `go mod init realchat`, add `cmd/agent/main.go` and a `.gitignore`, and verify `go build ./...` succeeds
- [x] 1.2 Implement `internal/llm` configuration from `GROQ_API_KEY`, `LLM_BASE_URL` and `LLM_MODEL`, and verify a unit test covers the missing-key error and a custom base URL
- [x] 1.3 Implement a single non-tool chat request with the standard library HTTP client and a timeout, and verify a test against `httptest.NewServer` covers a successful reply, a non-2xx error and a timeout
- [x] 1.4 Make `cmd/agent` print the reply for a hard-coded prompt, and verify it prints a real Groq answer with a valid key

## 2. M1: Messages and tool-call types

- [x] 2.1 Define message, tool definition and tool-call types matching the wire format, with `role`, `tool_calls` and `tool_call_id`, and verify JSON round-trip tests against sample provider payloads
- [x] 2.2 Extend the client to send a message history and tool definitions and to return tool calls, and verify an `httptest` test shows ordering, roles and tool-call decoding
- [x] 2.3 Add a minimal multi-turn chat in `cmd/agent` that keeps history, and verify a follow-up question uses the earlier turn

## 3. M2: Notes tools

- [x] 3.1 Add the sample corpus under `testdata/notes` with a subfolder and a file containing unique facts, and verify the files exist
- [x] 3.2 Implement the notes-root path resolver covering `..`, absolute paths, drive letters and symlinks, and verify table tests for every traversal case pass
- [x] 3.3 Implement `list_files`, `read_file` and `search_text` with sorted output, line numbers, case-insensitive search and truncation notes, and verify unit tests cover each scenario in the notes-tools spec
- [x] 3.4 Add the tool registry with name, description, JSON schema and dispatch by name, and verify a test that all three definitions are produced and that dispatch of an unknown name returns an error

## 4. M3: Agent loop

- [x] 4.1 Define the model interface and event types in `internal/agent`, and verify the package compiles with a scripted fake model in tests
- [x] 4.2 Implement the loop with the tool cycle, multiple calls per reply and direct answers, and verify tests with the fake model for no-tool, one-tool and two-tool replies
- [x] 4.3 Add the iteration cap and error-as-result handling for tool errors, unknown tools and invalid arguments, and verify tests for each case, including that the run continues
- [x] 4.4 Verify event order and nil-handler behavior with tests
- [x] 4.5 Make `cmd/agent` run the agent with tool activity printed from events, and verify a real question about the sample notes calls a tool and prints an answer

## 5. M4: CLI polish

- [x] 5.1 Add flags for the notes folder, model and iteration limit, and verify a missing folder exits non-zero with a message naming it
- [x] 5.2 Add one-shot mode with a question argument and the interactive mode with history, and verify exit status 0 for both, including on end of input
- [x] 5.3 Send configuration and iteration-limit errors to standard error with a non-zero exit in one-shot mode, and verify with a missing key and a forced cap of 1
- [x] 5.4 Add a README with setup, environment variables, and run commands, and verify the documented commands run as written

## 6. M5: Eval runner

- [x] 6.1 Define the eval case format in `testdata/eval.json` with 10 to 15 questions answerable from the sample notes, and verify the loader test rejects a malformed case
- [x] 6.2 Implement case checking for expected substrings and the required tool, and verify unit tests for pass, wrong answer and missing tool
- [x] 6.3 Implement `cmd/eval` that runs every case, treats errors as failures, continues on error, prints per-case results and the pass rate, and exits non-zero on failure, and verify a test with the fake model for all-pass, some-fail and error cases
- [ ] 6.4 Run the eval against the real model, record the model name and baseline pass rate in the README, and verify the recorded number matches a fresh run within normal variation

## 7. Integration

- [x] 7.1 Run `go vet ./...` and `go test ./...` from a clean checkout and verify both pass without network access or an API key
- [x] 7.2 Run `openspec validate add-cli-agent --strict` and verify there are no errors
