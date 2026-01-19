#!/bin/bash
# ralph.sh
# Autonomous AI coding loop executor

# Function to display help
show_help() {
    cat << EOF
Ralph Wiggum - Autonomous AI Coding Loop Executor

USAGE:
    ./ralph.sh [OPTIONS] [ITERATIONS]

ARGUMENTS:
    ITERATIONS    Number of iterations to run (default: 10)

OPTIONS:
    -h, --help    Show this help message
    -v            Enable verbose/debug output

DESCRIPTION:
    This script runs an autonomous AI coding loop using the opencode CLI.
    It executes the prompt defined in prompt.md, tracks progress in progress.txt,
    and manages tasks defined in tasks.md.

    The loop continues until all tasks are complete or the specified number 
    of iterations is reached.

FILES REQUIRED:
    - tasks.md     Task definitions and priorities
    - progress.txt Progress tracking
    - prompt.md    AI execution prompt

EXAMPLES:
    ./ralph.sh           # Run 10 iterations
    ./ralph.sh 5         # Run 5 iterations
    ./ralph.sh 20 -v     # Run 20 iterations with verbose output
    ./ralph.sh -h        # Show help

EOF
}

# Check for help flag
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    show_help
    exit 0
fi

set -e

# Check if opencode CLI is available
if ! command -v opencode &> /dev/null; then
    echo "Error: opencode CLI is not installed or not in PATH" >&2
    exit 1
fi

# Check if required files exist
required_files=("tasks.md" "progress.txt" "prompt.md")
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "Error: Required file $file not found" >&2
        exit 1
    fi
done

# Parse arguments
cnt=""
debug=""

for arg in "$@"; do
    case $arg in
        -v)
            debug='--print-logs'
            ;;
        [0-9]*)
            cnt=$arg
            ;;
    esac
done

if [ -z "$cnt" ]; then
    cnt=10
fi

# For each iteration, run Claude Code with the following prompt.
# This prompt is basic, we'll expand it later.
for ((i=1; i<$cnt; i++)); do
  result=$(opencode run $debug --model opencode/big-pickle "@tasks.md @progress.txt @prompt.md execute the prompt in prompt.md @tasks.md @progress.txt @prompt.md execute the prompt in prompt.md")

  echo "$result"

  if [[ "$result" == *"\u003cpromise\u003eCOMPLETE\u003c/promise\u003e"* ]]; then
    echo "PRD complete, exiting."
    exit 0
  fi
done