package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"realchat/internal/agent"
	"realchat/internal/eval"
	"realchat/internal/llm"
	"realchat/internal/tools"
)

// newModel is a variable so tests can substitute a fake model.
var newModel = func(cfg llm.Config) agent.Model { return llm.NewClient(cfg) }

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns the exit code: 0 all cases passed, 1 a case failed or setup failed, 2 bad usage.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(stderr)
	notesDir := fs.String("notes", "testdata/notes", "folder of markdown notes")
	casesPath := fs.String("cases", "testdata/eval.json", "eval cases file")
	model := fs.String("model", "", "model name (default: $"+llm.EnvModel+" or "+llm.DefaultModel+")")
	maxIter := fs.Int("max-iterations", agent.DefaultMaxIterations, "maximum model calls per question")
	delay := fs.Duration("delay", 0, "pause between cases, for example 2s, to stay under rate limits")
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

	cases, err := eval.LoadCases(*casesPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
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

	fmt.Fprintf(stdout, "Model: %s\nCases: %d\n\n", cfg.Model, len(cases))
	results := eval.Run(context.Background(), ag, cases, *delay, func(i int, r eval.Result) {
		fmt.Fprintln(stdout, eval.FormatResult(i, len(cases), r))
	})

	summary := eval.Summarize(results)
	fmt.Fprintf(stdout, "\n%s\n", summary)
	if summary.Passed != summary.Total {
		return 1
	}
	return 0
}
