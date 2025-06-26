package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"supertab/internal/ai"
	contextpkg "supertab/internal/context"
	"supertab/internal/history"

	"github.com/spf13/cobra"
)

// debugCmd represents the debug command
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug command to show collected context and history",
	Long: `Debug command that displays all the context information and history 
that would be sent to the AI, without making any API calls.`,
	RunE: runDebug,
}

func init() {
	rootCmd.AddCommand(debugCmd)

	// Command-specific flags
	debugCmd.Flags().Int("history-limit", 5, "number of recent history entries to show")
	debugCmd.Flags().Bool("json", false, "output in JSON format")
	debugCmd.Flags().Bool("debug-aliases", false, "show detailed alias collection debug info")
}

// runDebug executes the debug command logic
func runDebug(cmd *cobra.Command, args []string) error {
	historyLimit, _ := cmd.Flags().GetInt("history-limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	debugAliases, _ := cmd.Flags().GetBool("debug-aliases")

	// Debug alias collection if requested
	if debugAliases {
		shell := os.Getenv("SHELL")
		fmt.Printf("🔧 DEBUGGING ALIAS COLLECTION\n")
		fmt.Printf("Shell: %s\n", shell)

		// Test direct alias command
		fmt.Println("\n1. Testing direct 'alias' command:")
		if cmd := exec.Command("alias"); cmd != nil {
			output, err := cmd.Output()
			fmt.Printf("   Command: alias\n")
			fmt.Printf("   Error: %v\n", err)
			fmt.Printf("   Output length: %d\n", len(output))
			if len(output) > 0 {
				lines := strings.Split(string(output), "\n")
				fmt.Printf("   First few lines: %v\n", lines[:min(5, len(lines))])
			}
		}

		// Test shell -i -c alias
		fmt.Println("\n2. Testing shell interactive mode:")
		if cmd := exec.Command(shell, "-i", "-c", "alias"); cmd != nil {
			cmd.Env = os.Environ()
			output, err := cmd.Output()
			fmt.Printf("   Command: %s -i -c alias\n", shell)
			fmt.Printf("   Error: %v\n", err)
			fmt.Printf("   Output length: %d\n", len(output))
			if len(output) > 0 {
				lines := strings.Split(string(output), "\n")
				fmt.Printf("   First few lines: %v\n", lines[:min(5, len(lines))])

				// Test parsing
				aliases := make(map[string]string)
				parseAliasesDebug(string(output), aliases)
				fmt.Printf("   Parsed aliases count: %d\n", len(aliases))
				if len(aliases) > 0 {
					count := 0
					for name, command := range aliases {
						if count >= 3 {
							break
						}
						fmt.Printf("   Example: %s='%s'\n", name, command)
						count++
					}
				}
			}
		}

		// Test sourcing rc file
		fmt.Println("\n3. Testing rc file sourcing:")
		rcCommand := "source ~/.zshrc 2>/dev/null; alias 2>/dev/null"
		if cmd := exec.Command("sh", "-c", rcCommand); cmd != nil {
			cmd.Env = os.Environ()
			output, err := cmd.Output()
			fmt.Printf("   Command: sh -c \"%s\"\n", rcCommand)
			fmt.Printf("   Error: %v\n", err)
			fmt.Printf("   Output length: %d\n", len(output))
			if len(output) > 0 {
				lines := strings.Split(string(output), "\n")
				fmt.Printf("   First few lines: %v\n", lines[:min(5, len(lines))])
			}
		}

		return nil
	}

	// Collect context
	contextCollector := contextpkg.NewCollector()
	contextInfo := contextCollector.Collect()

	// Get recent history
	historyParser := history.NewParser()
	recentHistory, err := historyParser.GetRecentHistory(historyLimit)
	if err != nil {
		fmt.Printf("Warning: failed to get history: %v\n", err)
		recentHistory = []ai.HistoryEntry{}
	}

	if jsonOutput {
		// Output in JSON format
		debugInfo := map[string]interface{}{
			"context": contextInfo,
			"history": recentHistory,
		}

		jsonData, err := json.MarshalIndent(debugInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		fmt.Println(string(jsonData))
	} else {
		// Output in human-readable format
		fmt.Println("🔍 COLLECTED CONTEXT INFORMATION")
		fmt.Println("====================================================")

		fmt.Printf("User: %s\n", contextInfo.User)
		fmt.Printf("Directory: %s\n", contextInfo.Directory)
		fmt.Printf("Shell: %s\n", contextInfo.Shell)
		fmt.Printf("Terminal: %s\n", contextInfo.Terminal)
		fmt.Printf("System: %s\n", contextInfo.System)
		fmt.Printf("Platform: %s\n", contextInfo.Platform)
		fmt.Printf("DateTime: %s\n", contextInfo.DateTime.Format("2006-01-02 15:04:05"))

		// Git info
		if contextInfo.IsGitRepo {
			fmt.Printf("Git: Repository (branch: %s)\n", contextInfo.GitBranch)
		} else {
			fmt.Println("Git: Not a repository")
		}

		// Kubernetes info
		if contextInfo.K8sContext != nil && contextInfo.K8sContext.IsAvailable {
			fmt.Printf("Kubernetes: Available\n")
			fmt.Printf("  Context: %s\n", contextInfo.K8sContext.CurrentContext)
			fmt.Printf("  Namespace: %s\n", contextInfo.K8sContext.CurrentNamespace)
			if contextInfo.K8sContext.ClusterInfo != "" {
				fmt.Printf("  Cluster: %s\n", contextInfo.K8sContext.ClusterInfo)
			}
		} else {
			fmt.Println("Kubernetes: Not available")
		}

		// Aliases
		fmt.Printf("\n📝 SHELL ALIASES (%d found)\n", len(contextInfo.Aliases))
		fmt.Println("------------------------------")
		aliasCount := 0
		for name, command := range contextInfo.Aliases {
			if aliasCount >= 10 {
				fmt.Printf("... and %d more aliases\n", len(contextInfo.Aliases)-10)
				break
			}
			fmt.Printf("%s='%s'\n", name, command)
			aliasCount++
		}

		// History
		fmt.Printf("\n📚 RECENT COMMAND HISTORY (%d entries)\n", len(recentHistory))
		fmt.Println("----------------------------------------")
		for i, entry := range recentHistory {
			fmt.Printf("%d. [%s] %s", i+1, entry.Timestamp.Format("15:04:05"), entry.Command)
			if entry.ExitCode != 0 {
				fmt.Printf(" (exit: %d)", entry.ExitCode)
			}
			if entry.Duration != "" {
				fmt.Printf(" (%s)", entry.Duration)
			}
			fmt.Println()

			if entry.Output != "" && entry.Output != "[command output not available]" {
				fmt.Printf("   Output: %s\n", entry.Output)
			}
			if entry.ErrorOutput != "" {
				fmt.Printf("   Error: %s\n", entry.ErrorOutput)
			}
		}
	}

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseAliasesDebug parses alias command output (debug version)
func parseAliasesDebug(output string, aliases map[string]string) {
	lines := strings.Split(output, "\n")
	fmt.Printf("   Total lines to parse: %d\n", len(lines))

	for i, line := range lines {
		if i >= 5 { // Only debug first few lines
			break
		}
		line = strings.TrimSpace(line)
		fmt.Printf("   Line %d: '%s'\n", i, line)

		if strings.HasPrefix(line, "alias ") {
			// Parse format: alias name='command'
			parts := strings.SplitN(strings.TrimPrefix(line, "alias "), "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				aliases[name] = value
				fmt.Printf("   Parsed alias: %s='%s'\n", name, value)
			}
		} else if strings.Contains(line, "=") && !strings.HasPrefix(line, "alias ") {
			// Handle lines that start directly with alias definitions
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				if name != "" && value != "" {
					aliases[name] = value
					fmt.Printf("   Parsed direct alias: %s='%s'\n", name, value)
				}
			}
		}
	}
}
