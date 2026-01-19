# Ralph Wiggum Autonomous AI Coding Loop

This repository implements a Ralph Wiggum autonomous AI coding loop using the OpenCode CLI. Ralph runs in a loop, choosing tasks to work on autonomously until all work is complete.

## Overview

Ralph Wiggum is an approach to autonomous AI coding that lets an agent work unsupervised on a list of tasks. Instead of writing a new prompt for each phase of development, you run the same prompt in a loop and let the agent choose what to work on next.

## Files

- `ralph.go` - The main Go program that runs the autonomous coding loop
- `tasks.md` - Task definitions and priorities
- `progress.txt` - Progress tracking between iterations
- `prompt.md` - AI execution prompt
- `config.json` - Optional configuration file for customizing behavior
- `backups/` - Directory for progress.txt backups (created automatically)
- `multi_agent.go` - Multi-agent coordination and management functions
- `agents.go` - Agent type definitions and personality prompts
- `database.go` - SQLite database management for agent sessions
- `webhook.go` - Webhook notification system integration
- `webserver.go` - PWA web server for mobile management
- `web/` - Progressive Web App files for Android/mobile integration
- `validation.go` - Security and validation framework

## Setup

### Prerequisites

1. **Go 1.19+** - For building the ralph binary
2. **OpenCode CLI** - Required for the AI coding operations
3. **Git** - For version control and commit tracking
4. **SQLite** - For multi-agent session tracking (built-in with Go)

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
- `progress.txt` - Tracks progress (must exist, not created automatically)
- `prompt.md` - Contains the AI execution instructions

### Optional Files

- `config.json` - Custom configuration for prompt command, webhooks, and behavior

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
- `-v` - Enable verbose output (shows colorized opencode output)
- `-debug` - Enable debug output (implies -v, adds --print-logs to opencode)
- `-n N` - Set number of iterations (default: 10)
- `-backup` - Backup progress.txt before running
- `-restore` - Restore progress.txt from latest backup and exit
- `-agent <type>` - Use specific agent type (tester, debugger, researcher, etc.)
- `-use-agents` - Enable multi-agent mode
- `agents list` - List all available agent types
- `-show-progress` - Show agent progress for current project
- `-multi-agent` - Run coordinated multi-agent session
- `-web-server` - Start web server for PWA access
- `-web-port <port>` - Port for web server (default: 8080)

### Examples

```bash
# Run 5 iterations
./ralph -n 5

# Run 20 iterations with verbose output
./ralph -n 20 -v

# Run with debug output
./ralph -debug

# Backup progress and run
./ralph -backup -n 15

# Restore from backup
./ralph -restore

# Use specific agent type
./ralph -agent tester -n 10

# List available agents
./ralph agents list

# Show agent progress
./ralph -show-progress

# Run multi-agent session
./ralph -multi-agent -n 20

# Start web server for mobile access
./ralph -web-server -web-port 8080

# Show help
./ralph -h
```

## How It Works

Each iteration:
1. Reads the current tasks and progress
2. Runs OpenCode CLI with the configured prompt command
3. The AI chooses the highest priority task to work on
4. Implements the feature and runs feedback loops
5. Commits the changes and updates progress
6. Continues until all tasks are complete or iteration limit is reached
7. Optionally sends webhook notifications upon completion

The program uses the OpenAI base URL `http://100.83.162.29:1234` by default and the `opencode/big-pickle` model.

## Multi-Agent System

Giggum now supports a comprehensive multi-agent system with specialized AI personalities:

### Available Agent Types

- **tester** - Automated testing and quality assurance
- **debugger** - Bug identification and fixing
- **researcher** - Code analysis and investigation
- **backend-developer** - Server-side development
- **frontend-developer** - Client-side development
- **ux** - User experience design and optimization
- **ui** - User interface design
- **marketer** - Documentation and promotional content
- **feedbackseeker** - User feedback collection and analysis
- **simplifier** - Code simplification and refactoring
- **documentationwriter** - Technical documentation

### Multi-Agent Features

- **Agent Personalities**: Each agent has specialized prompts and behaviors
- **Session Tracking**: SQLite database tracks all agent sessions and progress
- **Coordinated Work**: Agents can work together on complex tasks
- **Progress Monitoring**: Real-time progress tracking across all agents
- **Database Persistence**: Agent work is stored in `.giggum.db` for recovery

### Usage Examples

```bash
# Use a specific agent for targeted work
./ralph -agent tester -n 5

# Run coordinated multi-agent session
./ralph -multi-agent -n 20

# List all available agents
./ralph agents list

# Check current project progress
./ralph -show-progress
```

## Progressive Web App (PWA)

Giggum includes a Progressive Web App for mobile and web-based agent management:

### Features

- **Mobile-First Interface**: Responsive design for Android and mobile devices
- **Real-time Agent Management**: View and control agents from your browser
- **Progress Tracking**: Monitor agent progress and sessions
- **Notification System**: Webhook-based notifications for completion events
- **PWA Installation**: Install as a native app on Android devices

### Starting the Web Server

```bash
# Start web server on default port 8080
./ralph -web-server

# Start on custom port
./ralph -web-server -web-port 3000
```

Access the PWA at `http://localhost:8080` in your browser. The app can be installed as a PWA on Android devices for native-like experience.

### Web Interface Features

- **Agent Dashboard**: View all available agents and their status
- **Session Management**: Start, stop, and monitor agent sessions
- **Progress Visualization**: Real-time progress charts and metrics
- **Mobile Optimized**: Touch-friendly interface for mobile devices

## Webhook Integration

Giggum supports webhook notifications for integration with external systems:

### Webhook Features

- **Completion Notifications**: Automatic webhook calls when iterations finish
- **Iteration Tracking**: Includes iteration count and completion status
- **Response Processing**: Optional processing of webhook responses into tasks
- **Custom Endpoints**: Configurable webhook URLs and payloads

### Configuration

Add webhook configuration to `config.json`:

```json
{
  "webhook_url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
  "wait_for_reply": false,
  "reply_prompt": "Enter your response (or press Enter to continue): ",
  "add_to_tasks": true
}
```

### Webhook Payload

When iterations complete, Giggum sends a POST request with:

```json
{
  "project": "your-project-name",
  "iterations": 10,
  "completed": true,
  "timestamp": "2026-01-19T08:30:00Z"
}
```

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

### Config File

Use the `config.json` file to customize Ralph's behavior:

```json
{
  "prompt_command": "@tasks.md @progress.txt @prompt.md Follow the instrunctions in prompt.md",
  "webhook_url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
  "wait_for_reply": false,
  "reply_prompt": "Enter your response (or press Enter to continue): ",
  "add_to_tasks": true
}
```

**Configuration Options:**
- `prompt_command`: Custom command to pass to OpenCode CLI (default: `@tasks.md @progress.txt @prompt.md Follow the instrunctions in prompt.md`)
- `webhook_url`: URL to send completion notifications to
- `wait_for_reply`: Whether to wait for and process webhook responses
- `reply_prompt`: Prompt text for webhook responses
- `add_to_tasks`: Whether to add webhook responses to tasks.md
- `use_agents`: Enable multi-agent mode by default
- `agent_type`: Default agent type to use
- `multi_agent`: Enable coordinated multi-agent sessions

### Progress Backup and Restore

Ralph includes automatic backup and restore functionality:

- **Backup**: Create timestamped backups of progress.txt before running
- **Restore**: Restore from the latest backup file

```bash
# Backup before running
./ralph -backup -n 10

# Restore from latest backup
./ralph -restore
```

Backup files are stored in the `backups/` directory with format `progress_YYYYMMDD_HHMMSS.txt`.

### Database and Persistence

Giggum uses SQLite for persistent storage of agent sessions and progress:

- **Session Storage**: All agent work is stored in `.giggum.db`
- **Progress Tracking**: Comprehensive progress tracking across all agents
- **Cross-Project**: Separate databases for each project directory
- **Recovery**: Resume work after interruptions using stored sessions
- **Analytics**: Built-in analytics for agent performance and productivity

### Security and Validation

The system includes comprehensive security and validation features:

- **Input Validation**: All user inputs are validated before processing
- **SQL Injection Protection**: Parameterized queries for database operations
- **Path Traversal Prevention**: File system access is restricted and validated
- **Command Injection Protection**: All shell commands are properly escaped
- **Configuration Validation**: Configuration files are validated before use
- **Error Handling**: Comprehensive error handling and logging
- **Safe Defaults**: Secure defaults for all configuration options

### Alternative Loop Types

The Ralph pattern can be adapted for different purposes:
- **Test Coverage Loop** - Increase test coverage to target percentage
- **Linting Loop** - Fix all linting errors automatically  
- **Entropy Loop** - Clean up code smells and unused code
- **Issue Triage** - Convert GitHub Issues to PRs
- **Multi-Agent Coordination** - Coordinated work across specialized agents

## Contributing

This is a demonstration of the Ralph Wiggum methodology. Feel free to adapt the scripts and prompts for your own projects.

## License

MIT License - see LICENSE file for details.

## Resources

- [Ralph Wiggum Article](https://ghuntley.com/ralph/) - Original methodology
- [Long-running Agent Research](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) - Academic background
- [OpenCode CLI](https://opencode.ai/) - AI coding tool used in this implementation
- [Multi-Agent Systems](https://en.wikipedia.org/wiki/Multi-agent_system) - Background on multi-agent coordination
- [Progressive Web Apps](https://web.dev/progressive-web-apps/) - PWA technology documentation
- [SQLite Database](https://sqlite.org/) - Embedded database for session storage