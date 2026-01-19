# ralph.sh
# Usage: ./ralph.sh {citerations}

set -e

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
  result=$(opencode run $debug --model opencode/big-pickle "@tasks.md @progress.txt @prompt.md execute the prompt in promot.md @tasks.md @progress.txt @prompt.md execute the prompt in promot.md")

  echo "$result"

  if [[ "$result" == *"\u003cpromise\u003eCOMPLETE\u003c/promise\u003e"* ]]; then
    echo "PRD complete, exiting."
    exit 0
  fi
done