package ai

import (
	"context"
	"fmt"
	"os"
)

// Client interface for AI providers
type Client interface {
	Complete(ctx context.Context, req CompletionRequest) (*Response, error)
	Predict(ctx context.Context, req PredictionRequest) (*Response, error)
}

// Config holds configuration for AI clients
type Config struct {
	Provider Provider
	APIKey   string
	BaseURL  string
	Debug    bool
}

// NewClient creates a new AI client based on provider configuration
func NewClient(config Config) (Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for provider %s", config.Provider)
	}

	switch config.Provider {
	case ProviderOpenAI:
		return NewOpenAIClient(config), nil
	case ProviderAnthropic:
		return NewAnthropicClient(config), nil
	case ProviderGemini:
		return NewGeminiClient(config), nil
	case ProviderGroq:
		return NewGroqClient(config), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// DetectProvider automatically detects the available AI provider based on environment variables
func DetectProvider() (Provider, string) {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return ProviderOpenAI, key
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return ProviderAnthropic, key
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return ProviderGemini, key
	}
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		return ProviderGroq, key
	}
	return "", ""
}
