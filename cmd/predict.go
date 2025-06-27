package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"supertab/internal/ai"
	contextpkg "supertab/internal/context"
	"supertab/internal/history"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// predictCmd represents the predict command
var predictCmd = &cobra.Command{
	Use:   "predict",
	Short: "Predict the next most likely command",
	Long: `Predict the next most likely command based on command history and current context.
This analyzes recent shell history and environmental context to suggest what you might want to run next.`,
	RunE:         runPredict,
	SilenceUsage: true, // Don't show usage on error
}

func init() {
	rootCmd.AddCommand(predictCmd)

	// Command-specific flags
	predictCmd.Flags().Int("history-limit", 5, "number of recent history entries to analyze")
	predictCmd.Flags().Duration("timeout", 10*time.Second, "request timeout")
}

// runPredict executes the predict command logic
func runPredict(cmd *cobra.Command, args []string) error {
	// Get configuration
	historyLimit, _ := cmd.Flags().GetInt("history-limit")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Detect AI provider if not specified
	providerName := viper.GetString("provider")
	apiKey := ""

	if providerName == "" {
		var provider ai.Provider
		provider, apiKey = ai.DetectProvider()
		if provider == "" {
			return fmt.Errorf("no AI provider found. Set one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, GROQ_API_KEY")
		}
		providerName = string(provider)
	} else {
		// Get API key for specified provider
		switch providerName {
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		case "gemini":
			apiKey = os.Getenv("GEMINI_API_KEY")
		case "groq":
			apiKey = os.Getenv("GROQ_API_KEY")
		default:
			return fmt.Errorf("unsupported provider: %s", providerName)
		}
	}

	if apiKey == "" {
		return fmt.Errorf("API key not found for provider %s", providerName)
	}

	// Create AI client
	config := ai.Config{
		Provider: ai.Provider(providerName),
		APIKey:   apiKey,
		Debug:    viper.GetBool("debug"),
	}

	client, err := ai.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	// Collect context
	contextCollector := contextpkg.NewCollector()
	contextInfo := contextCollector.Collect()

	// Get recent history
	historyParser := history.NewParser()
	recentHistory, err := historyParser.GetRecentHistory(historyLimit)
	if err != nil {
		if viper.GetBool("debug") {
			fmt.Fprintf(os.Stderr, "Warning: failed to get history: %v\n", err)
		}
		// Continue with empty history
		recentHistory = []ai.HistoryEntry{}
	}

	// Create prediction request
	req := ai.PredictionRequest{
		History: recentHistory,
		Context: contextInfo,
	}

	// Call AI service
	response, err := client.Predict(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get prediction: %w", err)
	}

	// For predict, we want the raw AI output without parsing
	// The AI should return properly formatted response with + or = prefix
	rawContent := response.Content

	// Simple validation: ensure response starts with + or =
	if len(rawContent) == 0 {
		return fmt.Errorf("empty response from AI")
	}

	firstChar := rawContent[0]
	if firstChar != '+' && firstChar != '=' {
		return fmt.Errorf("invalid response format: must start with + or =")
	}

	// Output the AI response directly
	fmt.Print(rawContent)

	return nil
}
