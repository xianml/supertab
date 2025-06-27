package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIClient implements the Client interface for OpenAI
type OpenAIClient struct {
	config Config
	client *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config Config) *OpenAIClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com"
	}
	return &OpenAIClient{
		config: config,
		client: &http.Client{},
	}
}

type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []choice  `json:"choices"`
	Error   *apiError `json:"error,omitempty"`
}

type choice struct {
	Message message `json:"message"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Complete generates command completions using OpenAI
func (c *OpenAIClient) Complete(ctx context.Context, req CompletionRequest) (*Response, error) {
	prompt := buildCompletionPrompt(req)
	return c.makeRequest(ctx, prompt)
}

// Predict generates command predictions using OpenAI
func (c *OpenAIClient) Predict(ctx context.Context, req PredictionRequest) (*Response, error) {
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

// makeRawRequest sends a request to OpenAI API and returns raw response
func (c *OpenAIClient) makeRawRequest(ctx context.Context, userPrompt string) (string, error) {
	reqBody := openAIRequest{
		Model: "gpt-4o-mini",
		Messages: []message{
			{Role: "system", Content: getSystemPrompt()},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Return raw content without parsing
	return strings.TrimSpace(apiResp.Choices[0].Message.Content), nil
}

// makeRequest sends a request to OpenAI API and processes the response
func (c *OpenAIClient) makeRequest(ctx context.Context, userPrompt string) (*Response, error) {
	reqBody := openAIRequest{
		Model: "gpt-4o-mini",
		Messages: []message{
			{Role: "system", Content: getSystemPrompt()},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return parseResponse(apiResp.Choices[0].Message.Content)
}

// parseResponse parses the AI response and determines the response type
func parseResponse(content string) (*Response, error) {
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	switch content[0] {
	case '+':
		return &Response{
			Type:    TypeCompletion,
			Content: content[1:],
		}, nil
	case '=':
		return &Response{
			Type:    TypeReplacement,
			Content: content[1:],
		}, nil
	default:
		// For predictions, assume it's a new command suggestion
		return &Response{
			Type:    TypePrediction,
			Content: content,
		}, nil
	}
}
