package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GroqClient implements the Client interface for Groq
type GroqClient struct {
	config Config
	client *http.Client
}

// NewGroqClient creates a new Groq client
func NewGroqClient(config Config) *GroqClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.groq.com/openai"
	}
	return &GroqClient{
		config: config,
		client: &http.Client{},
	}
}

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []groqChoice `json:"choices"`
	Error   *groqError   `json:"error,omitempty"`
}

type groqChoice struct {
	Message groqMessage `json:"message"`
}

type groqError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Complete generates command completions using Groq
func (c *GroqClient) Complete(ctx context.Context, req CompletionRequest) (*Response, error) {
	prompt := buildCompletionPrompt(req)
	return c.makeRequest(ctx, prompt)
}

// Predict generates command predictions using Groq
func (c *GroqClient) Predict(ctx context.Context, req PredictionRequest) (*Response, error) {
	prompt := buildPredictionPrompt(req)
	return c.makeRequest(ctx, prompt)
}

// makeRequest sends a request to Groq API and processes the response
func (c *GroqClient) makeRequest(ctx context.Context, userPrompt string) (*Response, error) {
	reqBody := groqRequest{
		Model: "llama-3.1-70b-versatile",
		Messages: []groqMessage{
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

	var apiResp groqResponse
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
