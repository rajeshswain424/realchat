package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"realchat/internal/agent"
	"realchat/internal/llm"
	"realchat/internal/tools"
)

// newModel is a variable so tests can substitute a fake model.
var newModel = func(cfg llm.Config) agent.Model { return llm.NewClient(cfg) }

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run returns the process exit code: 0 success, 1 runtime failure, 2 bad usage.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: agent [flags] [question]")
		fmt.Fprintln(stderr, "With no question, starts an interactive session. Type exit to quit.")
		fs.PrintDefaults()
	}
	notesDir := fs.String("notes", "testdata/notes", "folder of markdown notes")
	model := fs.String("model", "", "model name (default: $"+llm.EnvModel+" or "+llm.DefaultModel+")")
	maxIter := fs.Int("max-iterations", agent.DefaultMaxIterations, "maximum model calls per question")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *maxIter < 1 {
		fmt.Fprintln(stderr, "error: -max-iterations must be at least 1")
		return 2
	}

	root, err := tools.NewRoot(*notesDir)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	cfg, err := llm.ConfigFromEnv()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if *model != "" {
		cfg.Model = *model
	}

	ag := &agent.Agent{
		Model:         newModel(cfg),
		Tools:         tools.NewNotesRegistry(root),
		MaxIterations: *maxIter,
	}
	history := []llm.Message{{Role: llm.RoleSystem, Content: agent.SystemPrompt}}
	onEvent := eventPrinter(stderr)
	ctx := context.Background()

	if fs.NArg() > 0 {
		question := strings.Join(fs.Args(), " ")
		answer, _, err := ag.Run(ctx, history, question, onEvent)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintln(stdout, answer)
		return 0
	}

	return interactive(ctx, ag, history, onEvent, stdin, stdout, stderr)
}

func interactive(ctx context.Context, ag *agent.Agent, history []llm.Message, onEvent func(agent.Event), stdin io.Reader, stdout, stderr io.Writer) int {
	in := bufio.NewScanner(stdin)
	for {
		fmt.Fprint(stdout, "you> ")
		if !in.Scan() {
			if err := in.Err(); err != nil {
				fmt.Fprintln(stderr, "error:", err)
				return 1
			}
			return 0
		}
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		if line == "exit" {
			return 0
		}

		answer, updated, err := ag.Run(ctx, history, line, onEvent)
		if err != nil {
			// Keep the session alive and drop the failed turn from history.
			fmt.Fprintln(stderr, "error:", err)
			continue
		}
		history = updated
		fmt.Fprintln(stdout, answer)
	}
}

// eventPrinter shows tool activity on stderr so stdout carries only answers.
func eventPrinter(w io.Writer) func(agent.Event) {
	return func(e agent.Event) {
		switch e.Kind {
		case agent.EventToolStart:
			fmt.Fprintf(w, "  [tool] %s %s\n", e.ToolName, e.ToolArgs)
		case agent.EventToolResult:
			fmt.Fprintf(w, "  [result] %s\n", summarize(e.Result))
		case agent.EventLimitReached:
			fmt.Fprintln(w, "  [limit] iteration limit reached")
		}
	}
}

func summarize(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 100 {
		return string(r[:100]) + "..."
	}
	return s
}
