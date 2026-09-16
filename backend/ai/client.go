// Package ai talks to an OpenAI-compatible chat-completions endpoint so the game
// can (optionally) ask a model to referee an idea or author a new puzzle.
//
// Nothing here is required for the game to work: with no base URL and key the
// whole app runs on the deterministic engine alone. Credentials are supplied per
// request by the player's own browser, are never persisted and never logged.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// maxResponseBytes keeps a hostile or broken endpoint from exhausting memory.
	maxResponseBytes = 256 << 10
	// DefaultModel is used when the player does not name one.
	DefaultModel = "gpt-4o-mini"
	// DefaultTimeout bounds one AI call.
	DefaultTimeout = 30 * time.Second
	// MaxTimeout is the ceiling a player may ask for.
	MaxTimeout = 60 * time.Second
	// maxPromptRunes bounds what we send, so a huge world cannot blow up a request.
	maxPromptRunes = 8000
)

// Config is the endpoint the player configured in the app's settings panel.
type Config struct {
	BaseURL        string `json:"baseUrl"`
	APIKey         string `json:"apiKey"`
	Model          string `json:"model"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

// Validate checks the settings and fills in defaults.
func (c *Config) Validate() error {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.Model = strings.TrimSpace(c.Model)
	if c.BaseURL == "" {
		return errors.New("no AI base URL: enter your base URL, key and model in Settings, or switch AI features off")
	}
	if c.APIKey == "" {
		return errors.New("no AI key: enter your base URL, key and model in Settings, or switch AI features off")
	}
	parsed, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("the AI base URL is not a valid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("the AI base URL must start with http:// or https://")
	}
	if parsed.Host == "" {
		return errors.New("the AI base URL needs a host, for example http://127.0.0.1:3000/v1")
	}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = int(DefaultTimeout / time.Second)
	}
	if c.TimeoutSeconds > int(MaxTimeout/time.Second) {
		c.TimeoutSeconds = int(MaxTimeout / time.Second)
	}
	return nil
}

// Endpoint accepts either a bare base ("http://host/v1") or a full completions
// URL and returns the URL to POST to.
func (c Config) Endpoint() string {
	trimmed := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if strings.HasSuffix(trimmed, "/chat/completions") {
		return trimmed
	}
	return trimmed + "/chat/completions"
}

func (c Config) timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return DefaultTimeout
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// Error carries what actually went wrong so the HTTP layer can report it
// honestly instead of inventing a result.
type Error struct {
	Kind    string // "config" | "auth" | "server" | "protocol" | "timeout" | "unreachable"
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("%s (HTTP %d): %s", e.Kind, e.Status, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func newError(kind string, status int, format string, args ...any) *Error {
	return &Error{Kind: kind, Status: status, Message: fmt.Sprintf(format, args...)}
}

// Client performs chat-completion calls. The HTTP client is injectable so tests
// can point it at a local fake endpoint.
type Client struct {
	HTTP *http.Client
}

// NewClient returns a client with a single shared transport.
func NewClient() *Client {
	return &Client{HTTP: &http.Client{}}
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Chat sends one system+user exchange and returns the assistant's text.
func (c *Client) Chat(ctx context.Context, cfg Config, system, user string, temperature float64) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", newError("config", 0, "%s", err.Error())
	}
	if client := c.httpClient(); client == nil {
		return "", newError("config", 0, "no HTTP client configured")
	}
	if runes := []rune(system + user); len(runes) > maxPromptRunes {
		user = string([]rune(user)[:maxPromptRunes/2])
	}

	payload, err := json.Marshal(chatRequest{
		Model:       cfg.Model,
		Temperature: temperature,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return "", newError("protocol", 0, "could not encode the request: %v", err)
	}

	timeout := cfg.timeout()
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(callCtx, http.MethodPost, cfg.Endpoint(), bytes.NewReader(payload))
	if err != nil {
		return "", newError("config", 0, "could not build the request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	// The key is written straight into the header and never logged or echoed.
	request.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	request.Header.Set("User-Agent", "scribblebox/1.0")

	response, err := c.httpClient().Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || callCtx.Err() == context.DeadlineExceeded {
			return "", newError("timeout", 0, "the AI endpoint did not answer within %s", timeout)
		}
		return "", newError("unreachable", 0, "could not reach the AI endpoint: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return "", newError("unreachable", response.StatusCode, "could not read the AI response: %v", err)
	}
	if len(body) > maxResponseBytes {
		return "", newError("protocol", response.StatusCode, "the AI response was larger than %d bytes", maxResponseBytes)
	}

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return "", newError("auth", response.StatusCode, "the AI endpoint refused the key (%s)", describeAPIError(body, response.StatusCode))
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", newError("server", response.StatusCode, "the AI endpoint returned an error: %s", describeAPIError(body, response.StatusCode))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", newError("protocol", response.StatusCode, "the AI endpoint did not return JSON that could be read: %v", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", newError("server", response.StatusCode, "the AI endpoint reported an error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", newError("protocol", response.StatusCode, "the AI endpoint returned no choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", newError("protocol", response.StatusCode, "the AI endpoint returned an empty answer")
	}
	return content, nil
}

func (c *Client) httpClient() *http.Client {
	if c == nil || c.HTTP == nil {
		return &http.Client{}
	}
	return c.HTTP
}

// describeAPIError prefers the endpoint's own message but never echoes the key.
func describeAPIError(body []byte, status int) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if parsed.Error.Message != "" {
			return truncate(parsed.Error.Message, 200)
		}
		if parsed.Message != "" {
			return truncate(parsed.Message, 200)
		}
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return fmt.Sprintf("no message (%d)", status)
	}
	return truncate(text, 200)
}

func truncate(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}
