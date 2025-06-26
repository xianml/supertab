package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"supertab/internal/ai"
	contextpkg "supertab/internal/context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// completeCmd represents the complete command
var completeCmd = &cobra.Command{
	Use:   "complete [input]",
	Short: "Complete a partial shell command",
	Long: `Complete a partial shell command using AI.
The input can be provided as an argument or via stdin.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runComplete,
}

func init() {
	rootCmd.AddCommand(completeCmd)

	// Command-specific flags
	completeCmd.Flags().String("input", "", "input command to complete")
	completeCmd.Flags().Duration("timeout", 30*time.Second, "request timeout")
}

// runComplete executes the complete command logic
func runComplete(cmd *cobra.Command, args []string) error {
	// Get input from args, flag, or stdin
	var input string
	if len(args) > 0 {
		input = args[0]
	} else if flagInput, _ := cmd.Flags().GetString("input"); flagInput != "" {
		input = flagInput
	} else {
		return fmt.Errorf("input is required")
	}

	// Get timeout
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

	// Create completion request
	req := ai.CompletionRequest{
		Input:   input,
		Context: contextInfo,
	}

	// Call AI service
	response, err := client.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get completion: %w", err)
	}

	// Output the result based on response type
	switch response.Type {
	case ai.TypeCompletion:
		fmt.Printf("+%s", response.Content)
	case ai.TypeReplacement:
		fmt.Printf("=%s", response.Content)
	default:
		fmt.Print(response.Content)
	}

	return nil
}
