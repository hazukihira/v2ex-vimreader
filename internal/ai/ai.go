// Package ai talks to an OpenAI-compatible chat completions endpoint.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/config"
)

// ErrDisabled is returned when the AI integration is not configured.
var ErrDisabled = errors.New("ai summary is disabled")

// Client is a minimal OpenAI-compatible client.
type Client struct {
	cfg    config.AIConfig
	apiKey string
	hc     *http.Client
}

// New creates the AI client.
func New(cfg config.AIConfig, apiKey string) *Client {
	return &Client{
		cfg:    cfg,
		apiKey: apiKey,
		hc:     &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second},
	}
}

// Enabled reports whether summaries are configured.
func (c *Client) Enabled() bool {
	return c.cfg.Enabled && strings.TrimSpace(c.cfg.BaseURL) != "" && strings.TrimSpace(c.cfg.Model) != ""
}

// Model returns the configured model name.
func (c *Client) Model() string { return c.cfg.Model }

// Generate sends one chat completion request. When onDelta is not nil the
// response is streamed and onDelta is called for every text chunk.
func (c *Client) Generate(ctx context.Context, system, user string, onDelta func(string) error) (string, error) {
	if !c.Enabled() {
		return "", ErrDisabled
	}
	payload := map[string]any{
		"model": c.cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"stream":      onDelta != nil,
		"temperature": 0.3,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for k, v := range c.cfg.Headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("ai endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	if onDelta == nil {
		return decodeCompletion(resp.Body)
	}
	return streamCompletion(resp.Body, onDelta)
}

func decodeCompletion(r io.Reader) (string, error) {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Error != nil {
		return "", fmt.Errorf("ai error: %s", payload.Error.Message)
	}
	if len(payload.Choices) == 0 {
		return "", fmt.Errorf("ai returned no choices")
	}
	return payload.Choices[0].Message.Content, nil
}

func streamCompletion(r io.Reader, onDelta func(string) error) (string, error) {
	var out strings.Builder
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return out.String(), fmt.Errorf("ai error: %s", chunk.Error.Message)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			out.WriteString(choice.Delta.Content)
			if err := onDelta(choice.Delta.Content); err != nil {
				return out.String(), err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}
