#!/bin/bash

# ZSH Copilot v2 Installation Script

set -e

echo "🚀 Installing ZSH Copilot v2..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go first: https://golang.org/dl/${NC}"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if ! printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo -e "${RED}❌ Go version $REQUIRED_VERSION or higher is required. Current version: $GO_VERSION${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Go version $GO_VERSION found${NC}"

# Build the CLI tool
echo -e "${BLUE}📦 Building sug CLI tool...${NC}"
make build

if [ ! -f "build/sug" ]; then
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Build successful${NC}"

# Install the CLI tool
echo -e "${BLUE}📥 Installing sug CLI to /usr/local/bin...${NC}"
if ! sudo cp build/sug /usr/local/bin/; then
    echo -e "${RED}❌ Failed to install CLI tool. Please check permissions.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ CLI tool installed successfully${NC}"

# Verify installation
if command -v sug &> /dev/null; then
    echo -e "${GREEN}✓ sug CLI is available in PATH${NC}"
    sug --help | head -3
else
    echo -e "${YELLOW}⚠️ sug CLI not found in PATH. You may need to restart your shell.${NC}"
fi

# Check for API keys
echo -e "${BLUE}🔑 Checking for AI provider API keys...${NC}"

API_KEYS_FOUND=false

if [ -n "$OPENAI_API_KEY" ]; then
    echo -e "${GREEN}✓ OpenAI API key found${NC}"
    API_KEYS_FOUND=true
fi

if [ -n "$ANTHROPIC_API_KEY" ]; then
    echo -e "${GREEN}✓ Anthropic API key found${NC}"
    API_KEYS_FOUND=true
fi

if [ -n "$GEMINI_API_KEY" ]; then
    echo -e "${GREEN}✓ Gemini API key found${NC}"
    API_KEYS_FOUND=true
fi

if [ -n "$GROQ_API_KEY" ]; then
    echo -e "${GREEN}✓ Groq API key found${NC}"
    API_KEYS_FOUND=true
fi

if [ "$API_KEYS_FOUND" = false ]; then
    echo -e "${YELLOW}⚠️ No AI provider API keys found. Please set at least one:${NC}"
    echo "   export OPENAI_API_KEY=\"your-key\""
    echo "   export ANTHROPIC_API_KEY=\"your-key\""  
    echo "   export GEMINI_API_KEY=\"your-key\""
    echo "   export GROQ_API_KEY=\"your-key\""
fi

# Install zsh plugin
echo -e "${BLUE}🔌 Installing zsh plugin...${NC}"

# Detect zsh plugin directory
ZSH_CUSTOM="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"
PLUGIN_DIR="$ZSH_CUSTOM/plugins/zsh-copilot-v2"

if [ -d "$ZSH_CUSTOM" ]; then
    # Oh My Zsh detected
    mkdir -p "$PLUGIN_DIR"
    cp zsh-copilot-v2.plugin.zsh "$PLUGIN_DIR/"
    echo -e "${GREEN}✓ Plugin installed to Oh My Zsh: $PLUGIN_DIR${NC}"
    echo -e "${YELLOW}📝 Add 'zsh-copilot-v2' to your plugins list in ~/.zshrc${NC}"
else
    # Manual installation
    MANUAL_PLUGIN_PATH="$HOME/.zsh-copilot-v2.plugin.zsh"
    cp zsh-copilot-v2.plugin.zsh "$MANUAL_PLUGIN_PATH"
    echo -e "${GREEN}✓ Plugin installed to: $MANUAL_PLUGIN_PATH${NC}"
    echo -e "${YELLOW}📝 Add this line to your ~/.zshrc:${NC}"
    echo "   source $MANUAL_PLUGIN_PATH"
fi

# Copy example config
if [ ! -f "$HOME/.sug.yaml" ]; then
    cp .sug.yaml.example "$HOME/.sug.yaml"
    echo -e "${GREEN}✓ Example config copied to ~/.sug.yaml${NC}"
else
    echo -e "${BLUE}ℹ️ Config file ~/.sug.yaml already exists${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Installation completed successfully!${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo "1. Set up your AI provider API key"
echo "2. Add the plugin to your zsh configuration"
echo "3. Restart your shell or run: source ~/.zshrc"
echo "4. Use Ctrl+Z for completions and Down Arrow for predictions"
echo ""
echo -e "${BLUE}For help:${NC}"
echo "   sug --help"
echo "   sug complete --help"
echo "   sug predict --help"
echo ""
echo -e "${BLUE}Test the installation:${NC}"
echo "   sug complete \"git sta\"" 