#!/usr/bin/env zsh

# Default key bindings
(( ! ${+ZSH_COPILOT_KEY} )) &&
    typeset -g ZSH_COPILOT_KEY='^z'

(( ! ${+ZSH_COPILOT_PREDICT_KEY} )) &&
    typeset -g ZSH_COPILOT_PREDICT_KEY='^[[B'  # Down arrow key

# Configuration options
(( ! ${+ZSH_COPILOT_DEBUG} )) &&
    typeset -g ZSH_COPILOT_DEBUG=false

# CLI binary path (should be in PATH after installation)
(( ! ${+ZSH_COPILOT_CLI_PATH} )) &&
    typeset -g ZSH_COPILOT_CLI_PATH="sug"

# Provider override (optional, will auto-detect if not set)
(( ! ${+ZSH_COPILOT_AI_PROVIDER} )) &&
    typeset -g ZSH_COPILOT_AI_PROVIDER=""

# Timeout for AI requests
(( ! ${+ZSH_COPILOT_TIMEOUT} )) &&
    typeset -g ZSH_COPILOT_TIMEOUT="30s"

if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
    touch /tmp/zsh-copilot-v2.log
fi

# Function to show loading animation while waiting for AI response
function _show_loading_animation() {
    local pid=$1
    local interval=0.1
    local animation_chars=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
    local i=1

    cleanup() {
      kill $pid 2>/dev/null
      echo -ne "\e[?25h"
    }
    trap cleanup SIGINT
    
    while kill -0 $pid 2>/dev/null; do
        # Display current animation frame
        zle -R "${animation_chars[i]}"

        # Update index, make sure it starts at 1
        i=$(( (i + 1) % ${#animation_chars[@]} ))

        if [[ $i -eq 0 ]]; then
            i=1
        fi
        
        sleep $interval
    done

    echo -ne "\e[?25h"
    trap - SIGINT
}

# Function to call the backend CLI for command completion
function _fetch_completion() {
    local input="$1"
    local output_file="$2"
    
    # Prepare CLI arguments
    local cli_args=("$ZSH_COPILOT_CLI_PATH" "complete")
    
    if [[ -n "$ZSH_COPILOT_AI_PROVIDER" ]]; then
        cli_args+=(--provider "$ZSH_COPILOT_AI_PROVIDER")
    fi
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        cli_args+=(--debug)
    fi
    
    cli_args+=(--timeout "$ZSH_COPILOT_TIMEOUT")
    cli_args+=("$input")
    
    # Execute CLI command and capture output
    local result
    result=$("${cli_args[@]}" 2>&1)
    local exit_code=$?
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Called completion CLI\",\"input\":\"$input\",\"result\":\"$result\",\"exit_code\":\"$exit_code\",\"args\":\"${cli_args[*]}\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "$result" > "$output_file"
    else
        echo "Error: $result" > "$output_file.error"
    fi
}

# Function to call the backend CLI for command prediction
function _fetch_prediction() {
    local output_file="$1"
    
    # Prepare CLI arguments
    local cli_args=("$ZSH_COPILOT_CLI_PATH" "predict")
    
    if [[ -n "$ZSH_COPILOT_AI_PROVIDER" ]]; then
        cli_args+=(--provider "$ZSH_COPILOT_AI_PROVIDER")
    fi
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        cli_args+=(--debug)
    fi
    
    cli_args+=(--timeout "$ZSH_COPILOT_TIMEOUT")
    cli_args+=(--history-limit 5)
    
    # Execute CLI command and capture output
    local result
    result=$("${cli_args[@]}" 2>&1)
    local exit_code=$?
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Called prediction CLI\",\"result\":\"$result\",\"exit_code\":\"$exit_code\",\"args\":\"${cli_args[*]}\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "$result" > "$output_file"
    else
        echo "Error: $result" > "$output_file.error"
    fi
}

# Main AI suggestion function for command completion
function _suggest_ai() {
    # Clear any existing autosuggest
    _zsh_autosuggest_clear

    # Get input from current buffer
    local input=$(echo "${BUFFER:0:$CURSOR}" | tr '\n' ';')
    input=$(echo "$input" | sed 's/"/\\"/g')

    if [[ -z "$input" ]]; then
        echo "No input to complete"
        return 1
    fi

    # Prepare temporary files
    local suggestion_file="/tmp/zsh_copilot_suggestion_$$"
    local error_file="${suggestion_file}.error"
    
    # Clean up any existing temp files
    rm -f "$suggestion_file" "$error_file"

    # Start background process and show loading animation
    _fetch_completion "$input" "$suggestion_file" &
    local pid=$!
    
    _show_loading_animation $pid
    wait $pid
    local response_code=$?

    # Check for errors
    if [[ -f "$error_file" ]]; then
        echo "$(cat "$error_file")"
        rm -f "$suggestion_file" "$error_file"
        return 1
    fi

    if [[ ! -f "$suggestion_file" ]]; then
        echo "No suggestion available at this time. Please try again later."
        return 1
    fi

    local message=$(cat "$suggestion_file")
    rm -f "$suggestion_file" "$error_file"

    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Processing suggestion\",\"input\":\"$input\",\"message\":\"$message\"}" >> /tmp/zsh-copilot-v2.log
    fi

    # Process response based on first character
    local first_char=${message:0:1}
    local suggestion=${message:1:${#message}}
    
    if [[ "$first_char" == '=' ]]; then
        # Reset user input and replace with new command
        BUFFER=""
        CURSOR=0
        zle -U "$suggestion"
    elif [[ "$first_char" == '+' ]]; then
        # Append completion to current input
        _zsh_autosuggest_suggest "$suggestion"
    else
        # Fallback: treat as replacement
        BUFFER=""
        CURSOR=0
        zle -U "$message"
    fi
}

# Main AI prediction function for next command suggestion
function _predict_next_command() {
    # Only trigger if buffer is empty
    if [[ -n "$BUFFER" ]]; then
        # If there's already input, use normal down arrow behavior
        zle down-line-or-history
        return
    fi

    # Clear any existing autosuggest
    _zsh_autosuggest_clear

    # Prepare temporary files
    local prediction_file="/tmp/zsh_copilot_prediction_$$"
    local error_file="${prediction_file}.error"
    
    # Clean up any existing temp files
    rm -f "$prediction_file" "$error_file"

    # Start background process and show loading animation
    _fetch_prediction "$prediction_file" &
    local pid=$!
    
    _show_loading_animation $pid
    wait $pid
    local response_code=$?

    # Check for errors
    if [[ -f "$error_file" ]]; then
        if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
            echo "Prediction error: $(cat "$error_file")" >&2
        fi
        rm -f "$prediction_file" "$error_file"
        # Fallback to normal down arrow behavior
        zle down-line-or-history
        return
    fi

    if [[ ! -f "$prediction_file" ]]; then
        if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
            echo "No prediction available" >&2
        fi
        rm -f "$prediction_file" "$error_file"
        # Fallback to normal down arrow behavior
        zle down-line-or-history
        return
    fi

    local predicted_command=$(cat "$prediction_file")
    rm -f "$prediction_file" "$error_file"

    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Got prediction\",\"predicted_command\":\"$predicted_command\"}" >> /tmp/zsh-copilot-v2.log
    fi

    # Show the predicted command as a suggestion
    if [[ -n "$predicted_command" ]]; then
        _zsh_autosuggest_suggest "$predicted_command"
    else
        # Fallback to normal down arrow behavior
        zle down-line-or-history
    fi
}

# Information function
function zsh-copilot-v2() {
    echo "ZSH Copilot v2 (Frontend-Backend Architecture) is now active."
    echo ""
    echo "Key bindings:"
    echo "    - $ZSH_COPILOT_KEY: Get AI-powered command completion/suggestion"
    echo "    - $ZSH_COPILOT_PREDICT_KEY: Predict next command (when buffer is empty)"
    echo ""
    echo "Configuration:"
    echo "    - ZSH_COPILOT_KEY: Key binding for completions (current: $ZSH_COPILOT_KEY)"
    echo "    - ZSH_COPILOT_PREDICT_KEY: Key binding for predictions (current: $ZSH_COPILOT_PREDICT_KEY)"
    echo "    - ZSH_COPILOT_CLI_PATH: Path to sug CLI binary (current: $ZSH_COPILOT_CLI_PATH)"
    echo "    - ZSH_COPILOT_AI_PROVIDER: AI provider override (current: ${ZSH_COPILOT_AI_PROVIDER:-auto-detect})"
    echo "    - ZSH_COPILOT_TIMEOUT: AI request timeout (current: $ZSH_COPILOT_TIMEOUT)"
    echo "    - ZSH_COPILOT_DEBUG: Enable debug logging (current: $ZSH_COPILOT_DEBUG)"
    echo ""
    echo "Backend CLI status:"
    if command -v "$ZSH_COPILOT_CLI_PATH" &> /dev/null; then
        echo "    ✓ CLI binary found at: $(command -v "$ZSH_COPILOT_CLI_PATH")"
        echo "    ✓ Version: $("$ZSH_COPILOT_CLI_PATH" --version 2>/dev/null || echo "unknown")"
    else
        echo "    ✗ CLI binary not found in PATH: $ZSH_COPILOT_CLI_PATH"
        echo "    Please install the 'sug' CLI tool first"
    fi
}

# Register ZLE widgets and key bindings
zle -N _suggest_ai
zle -N _predict_next_command

bindkey "$ZSH_COPILOT_KEY" _suggest_ai
bindkey "$ZSH_COPILOT_PREDICT_KEY" _predict_next_command

# 检查函数是否定义
typeset -f _predict_next_command

# 检查是否注册为 ZLE widget
zle -l | grep predict 