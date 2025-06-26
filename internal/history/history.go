package history

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"supertab/internal/ai"
)

// Parser handles shell history parsing
type Parser struct{}

// NewParser creates a new history parser
func NewParser() *Parser {
	return &Parser{}
}

// GetRecentHistory retrieves recent command history entries with outputs
func (p *Parser) GetRecentHistory(limit int) ([]ai.HistoryEntry, error) {
	shell := os.Getenv("SHELL")

	var historyFile string
	switch {
	case strings.Contains(shell, "zsh"):
		historyFile = filepath.Join(os.Getenv("HOME"), ".zsh_history")
	case strings.Contains(shell, "bash"):
		historyFile = filepath.Join(os.Getenv("HOME"), ".bash_history")
	default:
		historyFile = filepath.Join(os.Getenv("HOME"), ".history")
	}

	entries, err := p.parseHistoryFile(historyFile, limit)
	if err != nil {
		return nil, err
	}

	// Try to get command outputs from enhanced history or cache
	p.enrichWithOutputs(entries)

	return entries, nil
}

// parseHistoryFile parses the shell history file and returns recent entries
func (p *Parser) parseHistoryFile(filename string, limit int) ([]ai.HistoryEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []ai.HistoryEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry := p.parseHistoryLine(line)
		if entry.Command != "" {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Ignore the last entry (current command) and return the previous 'limit' entries
	if len(entries) <= 1 {
		return []ai.HistoryEntry{}, nil
	}

	// Remove the last entry (current command)
	entries = entries[:len(entries)-1]

	// Return the last 'limit' entries from the remaining history
	start := len(entries) - limit
	if start < 0 {
		start = 0
	}

	return entries[start:], nil
}

// parseHistoryLine parses a single history line into a HistoryEntry
func (p *Parser) parseHistoryLine(line string) ai.HistoryEntry {
	entry := ai.HistoryEntry{
		Timestamp: time.Now(), // Default timestamp
		ExitCode:  0,          // Default exit code
	}

	// Handle zsh extended history format: : <timestamp>:<duration>;<command>
	if strings.HasPrefix(line, ": ") {
		parts := strings.SplitN(line, ";", 2)
		if len(parts) == 2 {
			// Parse timestamp from format ": 1234567890:0;"
			timestampPart := strings.TrimPrefix(parts[0], ": ")
			if colonIndex := strings.Index(timestampPart, ":"); colonIndex != -1 {
				timestampStr := timestampPart[:colonIndex]
				if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
					entry.Timestamp = time.Unix(timestamp, 0)
				}
			}
			entry.Command = parts[1]
		} else {
			entry.Command = line
		}
	} else {
		entry.Command = line
	}

	// Clean up the command
	entry.Command = strings.TrimSpace(entry.Command)

	return entry
}

// enrichWithOutputs attempts to enrich history entries with command outputs
func (p *Parser) enrichWithOutputs(entries []ai.HistoryEntry) {
	// TODO: Implement real command output retrieval from terminal history
	// This is a dummy implementation for now. To get real command outputs, we would need:
	// 1. Enhanced shell configuration to log outputs (e.g., script, tmux logging)
	// 2. Integration with terminal emulators that support history with outputs
	// 3. Custom shell hooks (precmd/preexec) to capture command results

	for i := range entries {
		entry := &entries[i]

		// For now, we'll leave outputs empty or provide dummy data for common commands
		entry.Output = ""
		entry.ErrorOutput = ""
		entry.ExitCode = 0
		entry.Duration = ""

		// Provide some context for common commands without executing them
		switch {
		case strings.HasPrefix(entry.Command, "ls"):
			entry.Output = "[command output not available]"
		case strings.HasPrefix(entry.Command, "git status"):
			entry.Output = "[git status output not available]"
		case strings.HasPrefix(entry.Command, "kubectl"):
			entry.Output = "[kubectl output not available]"
		}
	}
}

// Note: The following functions are placeholders for future implementation
// of real command output retrieval from terminal history

// TODO: Future enhancement - implement real command output retrieval
// Possible approaches:
// 1. Shell integration with logging (e.g., zsh precmd/preexec hooks)
// 2. Terminal emulator integration (tmux, screen logging)
// 3. System-level command auditing (auditd on Linux)
// 4. Custom shell wrapper that logs all command I/O
