#!/bin/bash

# Entrypoint script for giggum container
set -e

echo "Starting giggum container..."

# Set default working directory if not already set
cd /src

# Ensure required files exist or create them with defaults
for file in "tasks.md" "progress.txt" "prompt.md"; do
    if [ ! -f "$file" ]; then
        echo "Creating default $file..."
        case "$file" in
            "tasks.md")
                echo "# Tasks for giggum\n\n- Add your first task here\n" > "$file"
                ;;
            "progress.txt")
                echo "# Progress Log\n\n" > "$file"
                ;;
            "prompt.md")
                echo "# Giggum Execution Prompt\n\n1. Decide which task to work on next from @tasks.md\n2. Complete the task\n3. Append progress to progress.txt\n4. Make a git commit\n" > "$file"
                ;;
        esac
    fi
done

# Create default config.json if it doesn't exist
if [ ! -f "config.json" ]; then
    echo "Creating default config.json..."
    cat > config.json << EOF
{
    "prompt_command": "@tasks.md @progress.txt @prompt.md Follow the instructions in prompt.md",
    "webhook_url": "",
    "wait_for_reply": false,
    "reply_prompt": "Enter your response (or press Enter to continue): ",
    "add_to_tasks": true
}
EOF
fi

# Verify opencode CLI is available
if ! command -v opencode &> /dev/null; then
    echo "Error: opencode CLI not found. Installing..."
    npm install -g @opencode/cli
fi

# Set up Go environment for giggum
export GOPATH=/home/giggum/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

# Create GOPATH if it doesn't exist
mkdir -p $GOPATH

# Display environment information
echo "Environment:"
echo "  Working Directory: $(pwd)"
echo "  User: $(whoami)"
echo "  Go Version: $(go version 2>/dev/null || echo 'Go not available')"
echo "  Node Version: $(node --version 2>/dev/null || echo 'Node not available')"
echo "  Openencode Version: $(opencode --version 2>/dev/null || echo 'Opencode not available')"
echo "  OPENAI_BASE_URL: ${OPENAI_BASE_URL:-not set}"

# Check if we have required files
echo "Required files:"
for file in "tasks.md" "progress.txt" "prompt.md"; do
    if [ -f "$file" ]; then
        size=$(wc -c < "$file")
        echo "  ✓ $file ($size bytes)"
    else
        echo "  ✗ $file (missing)"
    fi
done

# If the first argument is "giggum", run it with remaining args
if [ "$1" = "giggum" ]; then
    echo "Starting giggum..."
    shift
    exec giggum "$@"
else
    # If no command specified, default to giggum
    if [ $# -eq 0 ]; then
        echo "No command specified, starting giggum..."
        exec giggum
    else
        # Execute the provided command
        echo "Executing command: $*"
        exec "$@"
    fi
fi