// Package gemini is a direct HTTP client for the Google Gemini
// generateContent API, used as the Krishi Mitra chatbot backend.
//
// It replaces the local Ollama runtime. The two clients are interchangeable
// from the handler's point of view: same inputs (system prompt, prior history,
// current message), same output (a trimmed reply), and the same error
// semantics, so the 502 branch and its wording are unchanged.
package gemini

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

// DefaultBaseURL is the Generative Language API root.
const DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// ErrUnreachable is returned when Gemini cannot be contacted or refuses the
// request. The handler maps it to the same 502 the Ollama path used.
var ErrUnreachable = errors.New("gemini unreachable")

// UnreachableMessage is what the user sees when the model cannot be reached.
//
// The Ollama wording named a local command the user could run; that is useless
// for a hosted API, so this one says what actually needs checking.
const UnreachableMessage = "Could not reach the AI service. Check GEMINI_API_KEY and network connectivity."

// EmptyReplyMessage matches the Ollama client's wording for a blank generation,
// because the frontend surfaces it verbatim.
const EmptyReplyMessage = "I could not process that. Please try asking again."

// Message is one chat turn, in the same shape the chatbot memory stores.
type Message struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// Client talks to the Gemini API.
type Client struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	// ThinkingBudget caps reasoning tokens on the 2.5 models. See New.
	ThinkingBudget int
	HTTP           *http.Client
}

// New returns a client. A generous timeout: generation is not fast, and the
// request already carries the caller's context for cancellation.
func New(apiKey, model, baseURL string, temperature float64, maxTokens, thinkingBudget int) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &Client{
		APIKey:         apiKey,
		Model:          model,
		BaseURL:        strings.TrimRight(baseURL, "/"),
		Temperature:    temperature,
		MaxTokens:      maxTokens,
		ThinkingBudget: thinkingBudget,
		HTTP:           &http.Client{Timeout: 120 * time.Second},
	}
}

// Enabled reports whether an API key is configured.
func (c *Client) Enabled() bool { return c != nil && c.APIKey != "" }

// ── wire types ──────────────────────────────────────────────────────────────

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type thinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type generationConfig struct {
	Temperature     float64         `json:"temperature"`
	MaxOutputTokens int             `json:"maxOutputTokens"`
	ThinkingConfig  *thinkingConfig `json:"thinkingConfig,omitempty"`
}

type generateRequest struct {
	SystemInstruction *content         `json:"system_instruction,omitempty"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type generateResponse struct {
	Candidates []struct {
		Content      content `json:"content"`
		FinishReason string  `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Chat sends the system prompt, the prior history and the current message.
//
// Two shape differences from the Ollama API that matter:
//
//   - Gemini has no "system" role. The system prompt goes in the separate
//     system_instruction field; putting it in contents would make the model
//     treat the farm statistics as something the user said.
//   - The assistant role is called "model". Sending "assistant" is rejected.
func (c *Client) Chat(ctx context.Context, systemPrompt string, history []Message, userMessage string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("%w: GEMINI_API_KEY is not set", ErrUnreachable)
	}

	contents := make([]content, 0, len(history)+1)
	for _, m := range history {
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "model"
		}
		contents = append(contents, content{Role: role, Parts: []part{{Text: m.Content}}})
	}
	contents = append(contents, content{Role: "user", Parts: []part{{Text: userMessage}}})

	req := generateRequest{
		Contents: contents,
		GenerationConfig: generationConfig{
			Temperature:     c.Temperature,
			MaxOutputTokens: c.MaxTokens,
		},
	}
	if systemPrompt != "" {
		req.SystemInstruction = &content{Parts: []part{{Text: systemPrompt}}}
	}
	// Thinking shares MaxOutputTokens with the visible answer, so on a 2.5
	// model a small budget can be spent entirely on reasoning and return an
	// empty reply with finishReason MAX_TOKENS. Sending thinkingConfig
	// explicitly keeps the whole budget for the answer unless an operator opts
	// back in via GEMINI_THINKING_BUDGET.
	if strings.Contains(c.Model, "2.5") || c.ThinkingBudget > 0 {
		req.GenerationConfig.ThinkingConfig = &thinkingConfig{ThinkingBudget: c.ThinkingBudget}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", c.BaseURL, c.Model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// The key goes in a header, not the query string, so it cannot leak into
	// proxy logs or browser history.
	httpReq.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}

	var out generateResponse
	if jsonErr := json.Unmarshal(raw, &out); jsonErr != nil {
		return "", fmt.Errorf("%w: malformed response (%d): %s",
			ErrUnreachable, resp.StatusCode, truncate(string(raw), 200))
	}
	if resp.StatusCode != http.StatusOK {
		msg := out.Error.Message
		if msg == "" {
			msg = truncate(string(raw), 200)
		}
		return "", fmt.Errorf("%w: gemini returned %d: %s", ErrUnreachable, resp.StatusCode, msg)
	}
	if out.PromptFeedback.BlockReason != "" {
		// A safety block is a refusal, not a transport failure — surface it as
		// a normal reply rather than a 502 the user cannot act on.
		return "", fmt.Errorf("the assistant declined to answer that (%s)",
			strings.ToLower(out.PromptFeedback.BlockReason))
	}
	if len(out.Candidates) == 0 {
		return "", errors.New(EmptyReplyMessage)
	}

	var sb strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	reply := strings.TrimSpace(sb.String())
	if reply == "" {
		return "", errors.New(EmptyReplyMessage)
	}
	return reply, nil
}

// Health checks the API key and model are usable, without generating anything.
func (c *Client) Health(ctx context.Context) error {
	if !c.Enabled() {
		return fmt.Errorf("%w: GEMINI_API_KEY is not set", ErrUnreachable)
	}
	url := fmt.Sprintf("%s/models/%s", c.BaseURL, c.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%w: status %d: %s", ErrUnreachable, resp.StatusCode, string(raw))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
