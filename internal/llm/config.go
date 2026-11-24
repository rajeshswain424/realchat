package llm

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	EnvAPIKey  = "GROQ_API_KEY"
	EnvBaseURL = "LLM_BASE_URL"
	EnvModel   = "LLM_MODEL"

	DefaultBaseURL = "https://api.groq.com/openai/v1"
	DefaultModel   = "openai/gpt-oss-120b"
)

// Config holds everything the client needs to reach an OpenAI-compatible endpoint.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string

	// Temperature is a pointer so an explicit 0 is sent instead of being omitted.
	Temperature         *float64
	MaxCompletionTokens int
	ReasoningEffort     string
	Timeout             time.Duration
}

// ConfigFromEnv builds a Config from environment variables, applying defaults.
func ConfigFromEnv() (Config, error) {
	key := os.Getenv(EnvAPIKey)
	if key == "" {
		return Config{}, fmt.Errorf("%w: set the %s environment variable", ErrMissingAPIKey, EnvAPIKey)
	}

	temp := 0.0
	cfg := Config{
		APIKey:              key,
		BaseURL:             DefaultBaseURL,
		Model:               DefaultModel,
		Temperature:         &temp,
		MaxCompletionTokens: 2048,
		ReasoningEffort:     "medium",
		Timeout:             60 * time.Second,
	}
	if v := os.Getenv(EnvBaseURL); v != "" {
		cfg.BaseURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv(EnvModel); v != "" {
		cfg.Model = v
	}
	return cfg, nil
}

var ErrMissingAPIKey = errors.New("missing API key")
