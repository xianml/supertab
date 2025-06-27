package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// AnthropicClient implements the Client interface for Anthropic
type AnthropicClient struct {
	config Config
	client *http.Client
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(config Config) *AnthropicClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	return &AnthropicClient{
		config: config,
		client: &http.Client{},
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
	Error   *anthropicError    `json:"error,omitempty"`
	Type    string             `json:"type"`
}

type anthropicContent struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete generates command completions using Anthropic
func (c *AnthropicClient) Complete(ctx context.Context, req CompletionRequest) (*Response, error) {
	prompt := buildCompletionPrompt(req)
	return c.makeRequest(ctx, prompt)
}

// Predict generates command predictions using Anthropic
func (c *AnthropicClient) Predict(ctx context.Context, req PredictionRequest) (*Response, error) {
	prompt := buildPredictionPrompt(req)

	// For prediction, we want the raw AI response without parsing
	rawContent, err := c.makeRawRequest(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Return the raw content directly
	return &Response{
		Type:    TypePrediction,
		Content: rawContent,
	}, nil
}

// makeRawRequest sends a request to Anthropic API and returns raw response
func (c *AnthropicClient) makeRawRequest(ctx context.Context, userPrompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     "claude-3-5-sonnet-latest",
		MaxTokens: 1000,
		System:    getSystemPrompt(),
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	if c.config.Debug {
		fmt.Fprintf(os.Stderr, "Debug: Request payload: %s\n", string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Type == "error" || apiResp.Error != nil {
		msg := "unknown error"
		if apiResp.Error != nil {
			msg = apiResp.Error.Message
		}
		return "", fmt.Errorf("API error: %s", msg)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	// Return raw content without parsing
	return strings.TrimSpace(apiResp.Content[0].Text), nil
}

// makeRequest sends a request to Anthropic API and processes the response
func (c *AnthropicClient) makeRequest(ctx context.Context, userPrompt string) (*Response, error) {
	reqBody := anthropicRequest{
		Model:     "claude-3-5-sonnet-latest",
		MaxTokens: 1000,
		System:    getSystemPrompt(),
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if c.config.Debug {
		fmt.Fprintf(os.Stderr, "Debug: Request payload: %s\n", string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Type == "error" || apiResp.Error != nil {
		msg := "unknown error"
		if apiResp.Error != nil {
			msg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("API error: %s", msg)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return parseResponse(apiResp.Content[0].Text)
}
