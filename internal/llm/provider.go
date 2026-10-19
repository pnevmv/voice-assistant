// Package llm defines the provider-neutral types the gateway exposes and the
// interface each backend (OpenAI-compatible, Anthropic, Ollama) implements.
// Services other than the gateway never import a provider implementation; they
// talk HTTP to the gateway (see internal/llm/client, added in M1).
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the gateway's internal request. Model is an alias from the
// gateway config ("fast", "smart", "extract") that resolves to an ordered
// provider chain, not a provider-specific model name.
type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature *float64
	MaxTokens   int
	Stop        []string
	// JSONSchema, when set, asks for structured output (used by deadline extraction).
	JSONSchema json.RawMessage
	// Metadata is free-form and used for accounting: caller, purpose, trace id.
	Metadata map[string]string
}

type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CostUSD          float64 `json:"cost_usd"`
}

type ChatResponse struct {
	Content      string
	Provider     string // concrete provider that served the call
	Model        string // concrete model name
	FinishReason string
	Usage        Usage
}

// Delta is one streamed piece. Usage and FinishReason are set on the last one.
type Delta struct {
	Text         string
	FinishReason string
	Usage        *Usage
}

// Provider is implemented once per backend. Implementations must be safe for
// concurrent use and must honour ctx cancellation. They do NOT retry or fall
// back; the gateway's router owns that policy.
type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// ChatStream yields deltas until completion. Breaking out of the loop must
	// release the underlying connection.
	ChatStream(ctx context.Context, req ChatRequest) iter.Seq2[Delta, error]
}

// Error is returned by providers so the router can decide retry vs fallback.
type Error struct {
	Provider  string
	Status    int  // HTTP status, 0 if transport-level
	Retryable bool // 429, 5xx, timeouts, connection resets
	Err       error
}

func (e *Error) Error() string { return "llm provider " + e.Provider + ": " + e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// ErrAllProvidersFailed is returned when the whole chain is exhausted.
var ErrAllProvidersFailed = errors.New("all providers failed")
