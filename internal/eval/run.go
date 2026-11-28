package eval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"realchat/internal/agent"
	"realchat/internal/llm"
)

// Result is the outcome of one case.
type Result struct {
	Case        Case
	Answer      string
	ToolsCalled []string
	Passed      bool
	Reason      string // why it failed; empty when passed
	Err         error  // agent error such as a network failure or the iteration cap
	Duration    time.Duration
}

// Run asks every case with a fresh conversation. Agent errors fail that case and the run continues.
// delay pauses between cases to stay under provider rate limits. onResult may be nil.
func Run(ctx context.Context, ag *agent.Agent, cases []Case, delay time.Duration, onResult func(i int, r Result)) []Result {
	results := make([]Result, 0, len(cases))
	for i, c := range cases {
		if i > 0 && delay > 0 {
			time.Sleep(delay)
		}
		r := runCase(ctx, ag, c)
		results = append(results, r)
		if onResult != nil {
			onResult(i, r)
		}
	}
	return results
}

func runCase(ctx context.Context, ag *agent.Agent, c Case) Result {
	r := Result{Case: c}
	start := time.Now()

	history := []llm.Message{{Role: llm.RoleSystem, Content: agent.SystemPrompt}}
	answer, _, err := ag.Run(ctx, history, c.Question, func(e agent.Event) {
		if e.Kind == agent.EventToolStart {
			r.ToolsCalled = append(r.ToolsCalled, e.ToolName)
		}
	})
	r.Duration = time.Since(start)
	r.Answer = answer

	if err != nil {
		r.Err = err
		r.Reason = "error: " + err.Error()
		return r
	}
	r.Passed, r.Reason = Check(c, answer, r.ToolsCalled)
	return r
}

// Summary counts passes across a run.
type Summary struct {
	Passed int
	Total  int
}

func Summarize(results []Result) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		if r.Passed {
			s.Passed++
		}
	}
	return s
}

// Rate is the pass rate as a percentage.
func (s Summary) Rate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Passed) * 100 / float64(s.Total)
}

func (s Summary) String() string {
	return fmt.Sprintf("Passed: %d/%d (%.1f%%)", s.Passed, s.Total, s.Rate())
}

// FormatResult renders one result for the terminal. Failures include the reason and the actual answer.
func FormatResult(i, n int, r Result) string {
	status := "PASS"
	if !r.Passed {
		status = "FAIL"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s [%d/%d] %s (%s)", status, i+1, n, r.Case.Question, r.Duration.Round(time.Millisecond))
	if !r.Passed {
		fmt.Fprintf(&b, "\n     reason: %s", r.Reason)
		if r.Err == nil {
			fmt.Fprintf(&b, "\n     answer: %s", oneLine(r.Answer))
		}
	}
	return b.String()
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if rs := []rune(s); len(rs) > 300 {
		return string(rs[:300]) + "..."
	}
	return s
}
