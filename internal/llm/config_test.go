package llm

import (
	"errors"
	"strings"
	"testing"
)

func TestConfigFromEnvMissingKey(t *testing.T) {
	t.Setenv(EnvAPIKey, "")

	_, err := ConfigFromEnv()
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("want ErrMissingAPIKey, got %v", err)
	}
	if !strings.Contains(err.Error(), EnvAPIKey) {
		t.Fatalf("error should name %s, got %q", EnvAPIKey, err)
	}
}

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv(EnvAPIKey, "k")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvModel, "")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != DefaultBaseURL || cfg.Model != DefaultModel {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestConfigFromEnvCustomBaseURL(t *testing.T) {
	t.Setenv(EnvAPIKey, "k")
	t.Setenv(EnvBaseURL, "http://localhost:11434/v1/")
	t.Setenv(EnvModel, "llama3")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("base URL = %q", cfg.BaseURL)
	}
	if cfg.Model != "llama3" {
		t.Fatalf("model = %q", cfg.Model)
	}
}
