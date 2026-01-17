# ralph.sh
# Usage: ./ralph.sh {citerations}

set -e

cnt = $1
if [ -z "$cnt" ]; then
  cnt = 10
fi

# For each iteration, run Claude Code with the following prompt.
# This prompt is basic, we'll expand it later.
for ((i=1; i<$cnt; i++)); do
  result=$(opencode run --model github-copilot/gpt-5-mini \\
"@wiggum.md @progress.txt \\
1. Decide which task to work on next. \\
This should be the one YOU decide has the highest priority, \\
- not necessarily the first in the list. \\
2. Check any feedback loops, such as types and tests. \\
3. Append your progress to the progress.txt file. \\
4. Make a git commit of that feature. \\
ONLY WORK ON A SINGLE FEATURE. \\
If, while implementing the feature, you notice that all work \\
is complete, output \u003cpromise\u003eCOMPLETE\u003c/promise\u003e. \\
")

  echo "$result"

  if [[ "$result" == *"\u003cpromise\u003eCOMPLETE\u003c/promise\u003e"* ]]; then
    echo "PRD complete, exiting."
    exit 0
  fi
done