# ZSH Copilot v2 - AI-Powered Shell Assistant

A modern, frontend-backend architecture for AI-powered shell command completion and prediction.

## Architecture

This project has been redesigned with a clean separation between frontend and backend:

- **Backend (`sug` CLI)**: Go-based command-line tool that handles AI interactions
- **Frontend (zsh plugin)**: Lightweight zsh script that provides UI and calls the backend

## Features

### 🚀 Command Completion
- AI-powered command completion using `Ctrl+Z`
- Supports both completion (`+`) and replacement (`=`) modes
- Context-aware suggestions based on current directory, git status, etc.

### 🔮 Command Prediction
- Predicts next command using `Down Arrow` key (when buffer is empty)
- Analyzes recent shell history and current context
- Smart suggestions based on usage patterns

### 🤖 Multi-AI Provider Support
- **OpenAI** GPT models
- **Anthropic** Claude models  
- **Google Gemini** models
- **Groq** models
- Auto-detection of available API keys

### 🎯 Smart Context Collection
- Current directory and git repository status
- Platform information (macOS, Linux, Windows)
- Shell and terminal information
- Recent command history with outputs
- User and system environment

## Quick Installation

```bash
# Clone and install
git clone <repository-url>
cd supertab
./install.sh
```

## Manual Installation

### 1. Install the Backend CLI

```bash
# Build the CLI tool
make build

# Install to system PATH
make install
```

### 2. Set up API Keys

Set at least one AI provider API key:

```bash
export OPENAI_API_KEY="your-openai-key"
# OR
export ANTHROPIC_API_KEY="your-anthropic-key"  
# OR
export GEMINI_API_KEY="your-gemini-key"
# OR
export GROQ_API_KEY="your-groq-key"
```

### 3. Install the zsh Plugin

```bash
# For Oh My Zsh users
mkdir -p ~/.oh-my-zsh/custom/plugins/zsh-copilot-v2
cp zsh-copilot-v2.plugin.zsh ~/.oh-my-zsh/custom/plugins/zsh-copilot-v2/
# Add 'zsh-copilot-v2' to plugins in ~/.zshrc

# Or source directly
echo "source $(pwd)/zsh-copilot-v2.plugin.zsh" >> ~/.zshrc

# Reload shell
source ~/.zshrc
```

## Usage

### Command Completion
1. Type a partial command: `git sta`
2. Press `Ctrl+Z`
3. Get AI-powered completion or replacement

### Command Prediction  
1. Ensure your command line is empty
2. Press `Down Arrow` key
3. Get a predicted next command based on your history

### CLI Usage
You can also use the backend directly:

```bash
# Complete a command
sug complete "git sta"

# Predict next command
sug predict --history-limit 5

# Use specific AI provider
sug complete "docker ru" --provider openai

# Enable debug mode
sug complete "ls -" --debug
```

## Configuration

### Environment Variables

```bash
# Frontend (zsh plugin) configuration
export ZSH_COPILOT_KEY='^z'                    # Completion key binding
export ZSH_COPILOT_PREDICT_KEY='^[[B'          # Prediction key binding  
export ZSH_COPILOT_CLI_PATH="sug"              # CLI binary path
export ZSH_COPILOT_AI_PROVIDER="openai"        # Force specific provider
export ZSH_COPILOT_TIMEOUT="30s"               # Request timeout
export ZSH_COPILOT_DEBUG=false                 # Debug logging

# Backend (CLI) configuration  
export OPENAI_API_KEY="your-key"               # OpenAI API key
export ANTHROPIC_API_KEY="your-key"            # Anthropic API key
export GEMINI_API_KEY="your-key"               # Gemini API key
export GROQ_API_KEY="your-key"                 # Groq API key
```

### Configuration File

Create `~/.sug.yaml` for persistent CLI configuration:

```yaml
provider: "openai"
debug: false
timeout: "30s"
```

## Development

### Building

```bash
# Install dependencies
make deps

# Build development version
make build-dev

# Run tests
make test

# Format and lint
make lint

# Full development workflow
make dev
```

### Project Structure

```
.
├── cmd/                    # CLI commands
│   ├── root.go            # Root command and configuration
│   ├── complete.go        # Completion command
│   └── predict.go         # Prediction command
├── internal/
│   ├── ai/                # AI provider implementations
│   │   ├── types.go       # Common types and interfaces
│   │   ├── client.go      # Client factory and interface
│   │   ├── openai.go      # OpenAI implementation
│   │   ├── anthropic.go   # Anthropic implementation
│   │   ├── gemini.go      # Gemini implementation
│   │   ├── groq.go        # Groq implementation
│   │   └── prompts.go     # AI prompts and prompt building
│   ├── context/           # System context collection
│   │   └── context.go     # Context collector implementation
│   └── history/           # Shell history parsing
│       └── history.go     # History parser implementation
├── zsh-copilot-v2.plugin.zsh  # New zsh plugin
├── zsh-copilot.plugin.zsh     # Original plugin (for reference)
├── main.go                     # CLI entry point
├── install.sh                  # Installation script
├── go.mod                      # Go module definition
├── Makefile                    # Build automation
└── README.md                   # This file
```

## Troubleshooting

### CLI Not Found
```bash
# Check if CLI is in PATH
which sug

# Check plugin configuration
echo $ZSH_COPILOT_CLI_PATH

# Reinstall CLI
make clean && make install
```

### API Errors
```bash
# Test CLI directly
sug complete "test" --debug

# Check API keys
env | grep -E "(OPENAI|ANTHROPIC|GEMINI|GROQ)_API_KEY"

# Check provider detection
sug complete "test" --provider auto
```

### Debug Mode
```bash
# Enable debug logging
export ZSH_COPILOT_DEBUG=true

# Check logs
tail -f /tmp/zsh-copilot-v2.log
```

## Migration from v1

The new version is backward compatible but offers improved performance and features:

1. Install the new backend CLI
2. Replace the old plugin with `zsh-copilot-v2.plugin.zsh`  
3. Optionally configure new features like command prediction

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with proper tests
4. Submit a pull request

## License

[MIT License](LICENSE)
