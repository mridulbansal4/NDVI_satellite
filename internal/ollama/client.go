// Package ollama is a direct HTTP client for the Ollama chat API.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §10.11, objective O5.
//
// LangChain is dropped entirely. The chain in chain.py is a system prompt plus
// history plus one user turn, which is a short HTTP client in Go.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrUnreachable is returned when Ollama cannot be contacted. The handler turns
// it into the exact 502 message the Python emits.
var ErrUnreachable = errors.New("ollama unreachable")

// UnreachableMessage is the verbatim text routes.py surfaces. The backticks
// around `ollama serve` are part of it.
const UnreachableMessage = "Could not reach Ollama. Run `ollama serve` and ensure the model is pulled."

// EmptyReplyMessage is what the Python returns when the model produces nothing.
const EmptyReplyMessage = "I could not process that. Please try asking again."

// Message is one chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client talks to a local Ollama server.
type Client struct {
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
	HTTP        *http.Client
}

// New returns a client with a sane timeout. Generation on CPU can be slow, so
// the timeout is generous rather than the usual few seconds.
func New(baseURL, model string, temperature float64, maxTokens int) *Client {
	return &Client{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		HTTP:        &http.Client{Timeout: 180 * time.Second},
	}
}

type chatRequest struct {
	Model    string    `json:"model"`
	Stream   bool      `json:"stream"`
	Messages []Message `json:"messages"`
	Options  options   `json:"options"`
}

type options struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

type chatResponse struct {
	Message Message `json:"message"`
	Error   string  `json:"error"`
}

// Chat sends the system prompt, the prior history and the current message, and
// returns the trimmed reply.
//
// Request shape per §10.11:
//
//	POST {base}/api/chat
//	{"model", "stream": false, "messages": [...], "options": {temperature, num_predict}}
//
// The reply is `.message.content`, trimmed.
func (c *Client) Chat(ctx context.Context, systemPrompt string, history []Message, userMessage string) (string, error) {
	msgs := make([]Message, 0, len(history)+2)
	msgs = append(msgs, Message{Role: "system", Content: systemPrompt})
	msgs = append(msgs, history...)
	msgs = append(msgs, Message{Role: "user", Content: userMessage})

	body, err := json.Marshal(chatRequest{
		Model:    c.Model,
		Stream:   false,
		Messages: msgs,
		Options:  options{Temperature: c.Temperature, NumPredict: c.MaxTokens},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: ollama returned %d: %s",
			ErrUnreachable, resp.StatusCode, truncate(string(raw), 200))
	}

	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("%w: malformed response: %v", ErrUnreachable, err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("%w: %s", ErrUnreachable, out.Error)
	}

	reply := strings.TrimSpace(out.Message.Content)
	if reply == "" {
		// The Python treats an empty generation as a failure, with its own
		// message and the same 502 status.
		return "", errors.New(EmptyReplyMessage)
	}
	return reply, nil
}

// Health reports whether the Ollama server answers at all.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrUnreachable, resp.StatusCode)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
