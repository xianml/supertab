#!/usr/bin/env zsh

# Default key bindings
(( ! ${+ZSH_COPILOT_KEY} )) &&
    typeset -g ZSH_COPILOT_KEY='^z'

(( ! ${+ZSH_COPILOT_PREDICT_KEY} )) &&
    typeset -g ZSH_COPILOT_PREDICT_KEY='^[[B'  # Down arrow key

# Configuration options
(( ! ${+ZSH_COPILOT_DEBUG} )) &&
    typeset -g ZSH_COPILOT_DEBUG=false

# Silent mode - don't show error messages to user
(( ! ${+ZSH_COPILOT_SILENT_ERRORS} )) &&
    typeset -g ZSH_COPILOT_SILENT_ERRORS=true

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

# Function to safely clean up temporary files
function _cleanup_temp_files() {
    rm -f /tmp/zsh_copilot_suggestion /tmp/zsh_copilot_prediction 2>/dev/null
    rm -f /tmp/zsh_copilot_error /tmp/zsh_copilot_prediction_error 2>/dev/null
}

# Function to safely restore terminal state
function _restore_terminal() {
    echo -ne "\e[?25h"  # Show cursor
    zle -R ""           # Clear any pending display
}

# Function to call the backend CLI for command completion
function _fetch_suggestions() {
    local input="$1"
    
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
    
    # Execute CLI command and capture only stdout (stderr contains config info)
    local result
    result=$("${cli_args[@]}" 2>/dev/null)
    local exit_code=$?
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        local error_output
        error_output=$("${cli_args[@]}" 2>&1 >/dev/null)
        echo "{\"date\":\"$(date)\",\"log\":\"Called completion CLI\",\"input\":\"$input\",\"result\":\"$result\",\"stderr\":\"$error_output\",\"exit_code\":\"$exit_code\",\"args\":\"${cli_args[*]}\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        # Clean up the response: remove trailing whitespace, newlines, and % characters
        result=$(echo "$result" | sed 's/[[:space:]]*$//' | tr -d '\n\r' | sed 's/%*$//')
        echo "$result" > /tmp/zsh_copilot_suggestion
    else
        if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
            echo "Backend CLI error: $result" > /tmp/zsh_copilot_error
        fi
        return 1
    fi
}

# Function to call the backend CLI for command prediction
function _fetch_prediction() {
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
    
    # Execute CLI command and capture only stdout
    local result
    result=$("${cli_args[@]}" 2>/dev/null)
    local exit_code=$?
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        local error_output
        error_output=$("${cli_args[@]}" 2>&1 >/dev/null)
        echo "{\"date\":\"$(date)\",\"log\":\"Called prediction CLI\",\"result\":\"$result\",\"stderr\":\"$error_output\",\"exit_code\":\"$exit_code\",\"args\":\"${cli_args[*]}\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        # Clean up the response
        result=$(echo "$result" | sed 's/[[:space:]]*$//' | tr -d '\n\r' | sed 's/%*$//')
        echo "$result" > /tmp/zsh_copilot_prediction
    else
        if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
            echo "Prediction CLI error: $result" > /tmp/zsh_copilot_prediction_error
        fi
        return 1
    fi
}

# Function to show loading animation while waiting for AI response
function _show_loading_animation() {
    local pid=$1
    local interval=0.1
    local animation_chars=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
    local i=1

    # Hide cursor and set up cleanup
    echo -ne "\e[?25l"
    
    cleanup() {
        kill $pid 2>/dev/null
        _restore_terminal
    }
    trap cleanup SIGINT SIGTERM
    
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

    _restore_terminal
    trap - SIGINT SIGTERM
}

# Function to show non-disruptive error message
function _show_error_message() {
    local message="$1"
    if [[ "$ZSH_COPILOT_SILENT_ERRORS" != 'true' ]]; then
        # Use POSTDISPLAY to show error without disrupting prompt
        POSTDISPLAY=" [AI Error: $message]"
        zle -R
        # Clear the error message after 2 seconds
        (sleep 2 && POSTDISPLAY="" && zle -R) &
    fi
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Error shown\",\"message\":\"$message\"}" >> /tmp/zsh-copilot-v2.log
    fi
}

# Main AI suggestion function for command completion
function _suggest_ai() {
    # Ensure terminal state is clean
    _restore_terminal
    _cleanup_temp_files
    
    # Clear any existing autosuggest
    _zsh_autosuggest_clear

    # Get input from current buffer
    local input=$(echo "${BUFFER:0:$CURSOR}" | tr '\n' ';')
    input=$(echo "$input" | sed 's/"/\\"/g')

    if [[ -z "$input" ]]; then
        if [[ "$ZSH_COPILOT_SILENT_ERRORS" != 'true' ]]; then
            _show_error_message "No input to complete"
        fi
        return 1
    fi

    # Fetch suggestions using v1's pattern to avoid job control issues
    read < <(_fetch_suggestions "$input" & echo $!)
    local pid=$REPLY

    _show_loading_animation $pid
    
    # Check if process still exists before waiting (avoid race condition)
    local response_code
    if kill -0 "$pid" 2>/dev/null; then
        wait "$pid"
        response_code=$?
    else
        # Process already finished, check if it succeeded by looking for output file
        if [[ -f /tmp/zsh_copilot_suggestion ]]; then
            response_code=0
        else
            response_code=1
        fi
    fi

    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Fetched suggestion\",\"input\":\"$input\",\"response_code\":\"$response_code\"}" >> /tmp/zsh-copilot-v2.log
    fi

    # Check for errors - fail silently if configured
    if [[ ! -f /tmp/zsh_copilot_suggestion || $response_code -ne 0 ]]; then
        _zsh_autosuggest_clear
        local error_msg="Service temporarily unavailable"
        if [[ -f /tmp/zsh_copilot_error && "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
            error_msg=$(cat /tmp/zsh_copilot_error 2>/dev/null || echo "$error_msg")
        fi
        _show_error_message "$error_msg"
        _cleanup_temp_files
        return 1
    fi

    local message=$(cat /tmp/zsh_copilot_suggestion)
    
    # If message is empty or just the prefix character, it's invalid
    if [[ -z "$message" || "$message" == "+" || "$message" == "=" ]]; then
        _show_error_message "No suggestion available"
        _cleanup_temp_files
        return 1
    fi

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
        # _zsh_autosuggest_suggest expects the full suggested command
        local full_suggestion="${BUFFER:0:$CURSOR}$suggestion"
        _zsh_autosuggest_suggest "$full_suggestion"
    else
        # Fallback: treat as replacement
        BUFFER=""
        CURSOR=0
        zle -U "$message"
    fi
    
    _cleanup_temp_files
}

# Main AI prediction function for next command suggestion
function _predict_next_command() {
    # Ensure terminal state is clean
    _restore_terminal
    _cleanup_temp_files
    
    # Debug output to see if function is being called
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"_predict_next_command called\",\"buffer\":\"$BUFFER\",\"buffer_length\":\"${#BUFFER}\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    if [[ -n "$BUFFER" ]]; then
        # If buffer is not empty, call default down-line-or-history
        if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
            echo "{\"date\":\"$(date)\",\"log\":\"Buffer not empty, calling down-line-or-history\"}" >> /tmp/zsh-copilot-v2.log
        fi
        zle down-line-or-history
        return
    fi
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Buffer empty, proceeding with prediction\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    _zsh_autosuggest_clear
    
    # Fetch prediction using v1's pattern
    read < <(_fetch_prediction & echo $!)
    local pid=$REPLY
    
    _show_loading_animation $pid
    
    # Check if process still exists before waiting (avoid race condition)
    local response_code
    if kill -0 "$pid" 2>/dev/null; then
        wait "$pid"
        response_code=$?
    else
        # Process already finished, check if it succeeded by looking for output file
        if [[ -f /tmp/zsh_copilot_prediction ]]; then
            response_code=0
        else
            response_code=1
        fi
    fi
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Fetched prediction\",\"response_code\":\"$response_code\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    # Check for errors - fail gracefully to normal behavior
    if [[ ! -f /tmp/zsh_copilot_prediction || $response_code -ne 0 ]]; then
        if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
            local error_msg="Prediction unavailable"
            if [[ -f /tmp/zsh_copilot_prediction_error ]]; then
                error_msg=$(cat /tmp/zsh_copilot_prediction_error 2>/dev/null || echo "$error_msg")
            fi
            echo "{\"date\":\"$(date)\",\"log\":\"Prediction failed\",\"error\":\"$error_msg\"}" >> /tmp/zsh-copilot-v2.log
        fi
        _cleanup_temp_files
        # Fallback to normal history navigation
        zle down-line-or-history
        return
    fi
    
    local predicted_command=$(cat /tmp/zsh_copilot_prediction)
    
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Got prediction\",\"predicted_command\":\"$predicted_command\"}" >> /tmp/zsh-copilot-v2.log
    fi
    
    if [[ -n "$predicted_command" ]]; then
        # Process prediction response based on first character (similar to _suggest_ai)
        local first_char=${predicted_command:0:1}
        local suggestion=${predicted_command:1:${#predicted_command}}
        
        if [[ "$first_char" == '=' ]]; then
            # Reset buffer and replace with new command
            BUFFER=""
            CURSOR=0
            zle -U "$suggestion"
        elif [[ "$first_char" == '+' ]]; then
            # Add prediction as suggestion (since buffer is empty, set POSTDISPLAY directly)
            POSTDISPLAY="$suggestion"
        else
            # Fallback: treat as complete suggestion
            _zsh_autosuggest_suggest "$predicted_command"
        fi
    else
        # Fallback to normal history navigation
        zle down-line-or-history
    fi
    
    _cleanup_temp_files
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
    echo "    - ZSH_COPILOT_SILENT_ERRORS: Hide error messages from user (current: $ZSH_COPILOT_SILENT_ERRORS)"
    echo ""
    echo "Error handling:"
    echo "    - Errors are handled gracefully to prevent shell disruption"
    echo "    - Failed operations fallback to normal shell behavior"
    echo "    - Set ZSH_COPILOT_SILENT_ERRORS=false to see error messages"
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

# Bind to multiple possible Down Arrow key sequences
bindkey "$ZSH_COPILOT_PREDICT_KEY" _predict_next_command  # ^[[B
bindkey "^[OB" _predict_next_command                      # Alternative Down Arrow
bindkey "^[[1B" _predict_next_command                     # Another variant

# Debug function to test key bindings
function _test_predict() {
    if [[ "$ZSH_COPILOT_DEBUG" == 'true' ]]; then
        echo "{\"date\":\"$(date)\",\"log\":\"Test function called\"}" >> /tmp/zsh-copilot-v2.log
    fi
}
zle -N _test_predict

# Temporarily bind F12 for testing prediction (easy to press)
bindkey "^[[24~" _test_predict 