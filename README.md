# Ralph Wiggum Autonomous AI Coding Loop

This repository implements a Ralph Wiggum autonomous AI coding loop using the OpenCode CLI. Ralph runs in a loop, choosing tasks to work on autonomously until all work is complete.

## Overview

Ralph Wiggum is an approach to autonomous AI coding that lets an agent work unsupervised on a list of tasks. Instead of writing a new prompt for each phase of development, you run the same prompt in a loop and let the agent choose what to work on next.

## Files

- `ralph.go` - The main Go program that runs the autonomous coding loop
- `tasks.md` - Task definitions and priorities
- `progress.txt` - Progress tracking between iterations
- `prompt.md` - AI execution prompt
- `PRD.json` - Example structured task tracking
- `wiggum.md` - Comprehensive documentation about Ralph Wiggum methodology

## Setup

### Prerequisites

1. **Go 1.19+** - For building the ralph binary
2. **OpenCode CLI** - Required for the AI coding operations
3. **Git** - For version control and commit tracking

### Installation

1. Clone this repository:
   ```bash
   git clone <repository-url>
   cd giggum
   ```

2. Build the ralph binary:
   ```bash
   go build -o ralph ralph.go
   ```

3. Ensure OpenCode CLI is installed and available in your PATH:
   ```bash
   opencode --version
   ```

### Required Files

Make sure these files exist in the directory:
- `tasks.md` - Contains your task list and priorities
- `progress.txt` - Tracks progress (will be created automatically)
- `prompt.md` - Contains the AI execution instructions

Example `tasks.md`:
```markdown
# My Project Tasks

## High Priority
- [ ] Implement user authentication
- [ ] Set up database schema

## Medium Priority
- [ ] Add API documentation
- [ ] Write unit tests
```

Example `prompt.md`:
```markdown
1. Decide which task to work on next from @tasks.md
2. Check any feedback loops (types, tests, linting)
3. Append your progress to progress.txt
4. Make a git commit of that feature
ONLY WORK ON A SINGLE FEATURE.
If all work is complete, output ✅ Complete ✅.
```

## Usage

### Basic Usage

Run the autonomous loop (default 10 iterations):
```bash
./ralph
```

### Options

- `-h, --help` - Show help message
- `-v` - Enable verbose/debug output
- `-n N` - Set number of iterations (default: 10)

### Examples

```bash
# Run 5 iterations
./ralph 5

# Run 20 iterations with verbose output
./ralph -n 20 -v

# Show help
./ralph -h
```

## How It Works

Each iteration:
1. Reads the current tasks and progress
2. Runs OpenCode CLI with the prompt from `prompt.md`
3. The AI chooses the highest priority task to work on
4. Implements the feature and runs feedback loops
5. Commits the changes and updates progress
6. Continues until all tasks are complete or iteration limit is reached

## Feedback Loops

The system relies on feedback loops to maintain code quality:
- **Type checking** - Catches type mismatches
- **Tests** - Prevents regressions
- **Linting** - Enforces code style
- **Pre-commit hooks** - Block bad commits

Add these to your prompt for better results:
```markdown
Before committing, run ALL feedback loops:
1. TypeScript: npm run typecheck (must pass)
2. Tests: npm run test (must pass) 
3. Lint: npm run lint (must pass)
Do NOT commit if any feedback loop fails.
```

## Best Practices

1. **Start with HITL (Human-in-the-Loop)** - Watch the first few runs to refine your prompt
2. **Keep tasks small** - One logical change per commit works best
3. **Prioritize risky work** - Tackle architectural decisions first
4. **Define clear acceptance criteria** - Be specific about what "done" means
5. **Use progress tracking** - The progress.txt file helps maintain context between runs

## Safety

For overnight or long-running sessions, consider using Docker sandboxes:
```bash
docker sandbox run ./ralph -n 50
```

This isolates Ralph from your system files while still allowing it to work on the project.

## Configuration

### Structured Task Tracking

Use the `PRD.json` file for structured task tracking:

```json
{
  "category": "functional",
  "description": "New chat button creates a fresh conversation",
  "steps": [
    "Click the 'New Chat' button",
    "Verify a new conversation is created",
    "Check that chat area shows welcome state"
  ],
  "passes": false
}
```

### Alternative Loop Types

The Ralph pattern can be adapted for different purposes:
- **Test Coverage Loop** - Increase test coverage to target percentage
- **Linting Loop** - Fix all linting errors automatically  
- **Entropy Loop** - Clean up code smells and unused code
- **Issue Triage** - Convert GitHub Issues to PRs

## Contributing

This is a demonstration of the Ralph Wiggum methodology. Feel free to adapt the scripts and prompts for your own projects.

## License

MIT License - see LICENSE file for details.

## Resources

- [Ralph Wiggum Article](https://ghuntley.com/ralph/) - Original methodology
- [Long-running Agent Research](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) - Academic background
- [OpenCode CLI](https://opencode.ai/) - AI coding tool used in this implementation