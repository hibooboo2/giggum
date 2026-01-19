#!/bin/bash
# ralph.sh
# Usage: ./ralph.sh {citerations}

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

cnt=$1
if [ -z "$cnt" ]; then
    cnt=10
fi

if [ "$2" == "-v" ]; then
    debug='--print-logs'
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