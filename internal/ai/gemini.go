package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GeminiClient implements the Client interface for Google Gemini
type GeminiClient struct {
	config Config
	client *http.Client
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(config Config) *GeminiClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com"
	}
	return &GeminiClient{
		config: config,
		client: &http.Client{},
	}
}

type geminiRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Complete generates command completions using Gemini
func (c *GeminiClient) Complete(ctx context.Context, req CompletionRequest) (*Response, error) {
	prompt := buildCompletionPrompt(req)
	return c.makeRequest(ctx, prompt)
}

// Predict generates command predictions using Gemini
func (c *GeminiClient) Predict(ctx context.Context, req PredictionRequest) (*Response, error) {
	prompt := buildPredictionPrompt(req)
	return c.makeRequest(ctx, prompt)
}

// makeRequest sends a request to Gemini API and processes the response
func (c *GeminiClient) makeRequest(ctx context.Context, userPrompt string) (*Response, error) {
	reqBody := geminiRequest{
		SystemInstruction: &geminiSystemInstruction{
			Parts: []geminiPart{{Text: getSystemPrompt()}},
		},
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: userPrompt}},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/gemini-1.5-flash-latest:generateContent?key=%s",
		c.config.BaseURL, c.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return parseResponse(apiResp.Candidates[0].Content.Parts[0].Text)
}
