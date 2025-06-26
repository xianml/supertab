package ai

import (
	"fmt"
	"strings"
)

// getSystemPrompt returns the base system prompt for all AI interactions
func getSystemPrompt() string {
	return ` You are a shell command completion assistant for backend developers or Site Reliability Engineering (SRE).
Your task is to either complete the command or provide a new command that you think the user is trying to type. 

1. COMMAND COMPLETION: When given a partial command, complete it or suggest a replacement.
2. COMMAND PREDICTION: When given command history and context, predict the next most likely command.

RESPONSE FORMAT RULES:
- For completions: prefix with '+' (e.g., "+mp" to complete "cd /t" -> "cd /tmp")
- For replacements: prefix with '=' (e.g., "=ls -la" to replace "list files")
- For predictions: prefix with '+' (e.g., "+kubectl -n <namespace> logs -f <failed pod>" to debug a failed pod)

CRITICAL REQUIREMENTS:
- Your response MUST be a single line without newlines
- Do not write any leading or trailing characters except if required for the completion to work
- make sure NO explanations, comments, or additional text
- Make sure commands are properly escaped and executable
- Make sure to only include the rest of the completion when completing a command.
- Consider the user's shell, OS, and current context
- For predictions, suggest commonly used commands based on patterns
- If the result command matches user's aliases, use the alias instead of the full command

When predicting next command, you should prioritize considering user's previous commands and their output. 
`
}

// buildCompletionPrompt builds a prompt for command completion
func buildCompletionPrompt(req CompletionRequest) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("INPUT: %s", req.Input))

	if req.Context.Directory != "" {
		parts = append(parts, fmt.Sprintf("DIRECTORY: %s", req.Context.Directory))
	}

	if req.Context.Platform != "" {
		parts = append(parts, fmt.Sprintf("PLATFORM: %s", req.Context.Platform))
	}

	if req.Context.IsGitRepo {
		gitInfo := "Git repository"
		if req.Context.GitBranch != "" {
			gitInfo += fmt.Sprintf(" (branch: %s)", req.Context.GitBranch)
		}
		parts = append(parts, fmt.Sprintf("GIT: %s", gitInfo))
	}

	// Add aliases information
	if len(req.Context.Aliases) > 0 {
		parts = append(parts, "ALIASES:")
		aliasCount := 0
		for name, command := range req.Context.Aliases {
			if aliasCount >= 10 { // Limit to most relevant aliases
				break
			}
			parts = append(parts, fmt.Sprintf("  %s='%s'", name, command))
			aliasCount++
		}
	}

	// Add Kubernetes context
	if req.Context.K8sContext != nil && req.Context.K8sContext.IsAvailable {
		k8sInfo := fmt.Sprintf("Kubernetes cluster connected (context: %s", req.Context.K8sContext.CurrentContext)
		if req.Context.K8sContext.CurrentNamespace != "" {
			k8sInfo += fmt.Sprintf(", namespace: %s", req.Context.K8sContext.CurrentNamespace)
		}
		k8sInfo += ")"
		parts = append(parts, fmt.Sprintf("K8S: %s", k8sInfo))
	}

	parts = append(parts, fmt.Sprintf("USER: %s", req.Context.User))
	parts = append(parts, fmt.Sprintf("SHELL: %s", req.Context.Shell))

	return strings.Join(parts, "\n")
}

// buildPredictionPrompt builds a prompt for command prediction
func buildPredictionPrompt(req PredictionRequest) string {
	var parts []string

	parts = append(parts, "TASK: Predict the next most likely command based on history and context.")

	// Add recent command history
	if len(req.History) > 0 {
		parts = append(parts, "\nRECENT HISTORY:")
		for i, entry := range req.History {
			timestamp := entry.Timestamp.Format("15:04:05")
			exitInfo := ""
			if entry.ExitCode != 0 {
				exitInfo = fmt.Sprintf(" (exit: %d)", entry.ExitCode)
			}

			// Show command with timing info
			cmdInfo := fmt.Sprintf("%d. [%s] %s%s", i+1, timestamp, entry.Command, exitInfo)
			if entry.Duration != "" {
				cmdInfo += fmt.Sprintf(" (%s)", entry.Duration)
			}
			parts = append(parts, cmdInfo)

			// Show output if available (truncated)
			if entry.Output != "" && entry.Output != "[command output not available]" {
				output := entry.Output
				if len(output) > 200 {
					lines := strings.Split(output, "\n")
					if len(lines) > 3 {
						lines = lines[:3]
						lines = append(lines, "...(truncated)")
					}
					output = strings.Join(lines, "\n")
				}
				parts = append(parts, fmt.Sprintf("   Output: %s", strings.TrimSpace(output)))
			}

			// Show error output if available
			if entry.ErrorOutput != "" {
				errorOutput := entry.ErrorOutput
				if len(errorOutput) > 100 {
					errorOutput = errorOutput[:100] + "...(truncated)"
				}
				parts = append(parts, fmt.Sprintf("   Error: %s", strings.TrimSpace(errorOutput)))
			}
		}
	}

	// Add current context information
	parts = append(parts, "\nCURRENT CONTEXT:")
	parts = append(parts, fmt.Sprintf("Directory: %s", req.Context.Directory))
	parts = append(parts, fmt.Sprintf("User: %s", req.Context.User))
	parts = append(parts, fmt.Sprintf("Platform: %s", req.Context.Platform))
	parts = append(parts, fmt.Sprintf("Shell: %s", req.Context.Shell))
	parts = append(parts, fmt.Sprintf("Time: %s", req.Context.DateTime.Format("2006-01-02 15:04:05")))

	// Add Git context
	if req.Context.IsGitRepo {
		gitInfo := "Yes"
		if req.Context.GitBranch != "" {
			gitInfo += fmt.Sprintf(" (branch: %s)", req.Context.GitBranch)
		}
		parts = append(parts, fmt.Sprintf("Git Repository: %s", gitInfo))
	} else {
		parts = append(parts, "Git Repository: No")
	}

	// Add Kubernetes context
	if req.Context.K8sContext != nil && req.Context.K8sContext.IsAvailable {
		k8sInfo := fmt.Sprintf("Yes (context: %s", req.Context.K8sContext.CurrentContext)
		if req.Context.K8sContext.CurrentNamespace != "" {
			k8sInfo += fmt.Sprintf(", namespace: %s", req.Context.K8sContext.CurrentNamespace)
		}
		k8sInfo += ")"
		parts = append(parts, fmt.Sprintf("Kubernetes: %s", k8sInfo))

		if req.Context.K8sContext.ClusterInfo != "" {
			parts = append(parts, fmt.Sprintf("Cluster: %s", req.Context.K8sContext.ClusterInfo))
		}
	} else {
		parts = append(parts, "Kubernetes: Not available")
	}

	// Add relevant aliases
	if len(req.Context.Aliases) > 0 {
		parts = append(parts, "\nAVAILABLE ALIASES:")
		aliasCount := 0
		for name, command := range req.Context.Aliases {
			if aliasCount >= 15 { // Show more aliases for prediction context
				break
			}
			parts = append(parts, fmt.Sprintf("  %s='%s'", name, command))
			aliasCount++
		}
	}

	parts = append(parts, "\nBased on the command history patterns, current context, available aliases, and Kubernetes environment, what command is the user most likely to run next?")
	parts = append(parts, "Consider:")
	parts = append(parts, "- Command execution patterns and failures")
	parts = append(parts, "- Directory context and git repository state")
	parts = append(parts, "- Kubernetes context and common operations")
	parts = append(parts, "- Available aliases that might be useful")
	parts = append(parts, "- Time of day and typical workflow patterns")

	return strings.Join(parts, "\n")
}
