package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

// Client talks to an OpenAI-compatible chat-completions endpoint.
type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// APIError is a non-2xx response from the provider.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("llm api error: status %d: %s", e.Status, e.Message)
}

// ErrTimeout is returned when the provider does not answer in time.
var ErrTimeout = errors.New("llm request timed out")

type chatRequest struct {
	Model               string           `json:"model"`
	Messages            []Message        `json:"messages"`
	Tools               []ToolDefinition `json:"tools,omitempty"`
	Temperature         *float64         `json:"temperature,omitempty"`
	MaxCompletionTokens int              `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string           `json:"reasoning_effort,omitempty"`
	Stream              bool             `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat sends the conversation and tool definitions and returns the model's reply message.
func (c *Client) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Message, error) {
	body, err := json.Marshal(chatRequest{
		Model:               c.cfg.Model,
		Messages:            messages,
		Tools:               tools,
		Temperature:         c.cfg.Temperature,
		MaxCompletionTokens: c.cfg.MaxCompletionTokens,
		ReasoningEffort:     c.cfg.ReasoningEffort,
	})
	if err != nil {
		return Message{}, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			return Message{}, ErrTimeout
		}
		return Message{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var er errorResponse
		msg := string(data)
		if json.Unmarshal(data, &er) == nil && er.Error.Message != "" {
			msg = er.Error.Message
		}
		return Message{}, &APIError{Status: resp.StatusCode, Message: msg}
	}

	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return Message{}, fmt.Errorf("decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return Message{}, errors.New("llm response contained no choices")
	}
	return out.Choices[0].Message, nil
}
