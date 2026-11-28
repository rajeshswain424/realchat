# RealChat

A notes agent written in Go. Ask a question about a folder of markdown files. An LLM decides which tools to call (`list_files`, `read_file`, `search_text`), reads what it needs, and answers. See [ROADMAP.md](ROADMAP.md) for where the project is going.

## Setup

Requires Go 1.22 or newer and a [Groq](https://console.groq.com) API key.

PowerShell:

```powershell
$env:GROQ_API_KEY = "your-key"
```

bash or zsh:

```bash
export GROQ_API_KEY="your-key"
```

The key is read only from the environment and is never written to disk or logged.

| Variable | Required | Default |
|----------|----------|---------|
| `GROQ_API_KEY` | yes | none |
| `LLM_BASE_URL` | no | `https://api.groq.com/openai/v1` |
| `LLM_MODEL` | no | `openai/gpt-oss-120b` |

## Run

Ask one question and exit:

```
go run ./cmd/agent "What is the API rate limit?"
```

Start an interactive session that remembers earlier turns. Type `exit` to quit:

```
go run ./cmd/agent
```

Tool calls are printed to stderr as they happen. Only the answer goes to stdout.

Flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `-notes` | `testdata/notes` | Folder of markdown notes the agent may read |
| `-model` | `$LLM_MODEL` or the default above | Model name |
| `-max-iterations` | `10` | Maximum model calls per question |

Point it at your own notes:

```
go run ./cmd/agent -notes C:\path\to\notes "What did I decide about auth?"
```

Exit codes: `0` success, `1` runtime failure (missing key, missing folder, network error, iteration limit), `2` bad flags. `go run` reports any non-zero code as `1`; build with `go build -o agent.exe ./cmd/agent` to see the real code.

## Develop

```
go vet ./...
go test ./...
```

Tests use a fake model and need neither network access nor an API key.

## Eval

The eval asks 14 fixed questions about the notes in `testdata/notes` and checks each answer for expected text, and for a required tool call where a case names one. It prints one line per case and a pass rate, and exits non-zero if any case fails.

```
go run ./cmd/eval
```

Flags: `-cases` (default `testdata/eval.json`), `-notes`, `-model`, `-max-iterations`, and `-delay` (for example `-delay 2s` to stay under rate limits). A case that hits a network error or the iteration limit counts as failed and the run continues.

Each case is `{"question": "...", "expect_contains": ["..."], "expect_tool": "optional"}`. Matching is case-insensitive and ignores typographic spaces and hyphens.

## Layout

```
cmd/agent/        command line program
cmd/eval/         eval runner
internal/llm/     OpenAI-compatible chat client, message and tool types
internal/agent/   the model-and-tools loop and its events
internal/tools/   notes tools, path sandbox, tool registry
internal/eval/    eval case loading, checking and reporting
testdata/notes/   sample notes used for manual runs and evals
testdata/eval.json  the eval cases
```
