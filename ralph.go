package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CLIArgs struct {
	help         bool
	iterations   int
	verbose      bool
	debug        bool
	backup       bool
	restore      bool
	agentType    string
	listAgents   bool
	showProgress bool
	multiAgent   bool
	taskCommand  string
}

func parseFlags() CLIArgs {
	var args CLIArgs
	flag.BoolVar(&args.help, "h", false, "Show help message")
	flag.BoolVar(&args.help, "help", false, "Show help message")
	flag.IntVar(&args.iterations, "n", 10, "Number of iterations to run")
	flag.BoolVar(&args.verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&args.debug, "debug", false, "Enable debug output (implies -v)")
	flag.BoolVar(&args.backup, "backup", false, "Backup progress before running")
	flag.BoolVar(&args.restore, "restore", false, "Restore progress from latest backup and exit")
	flag.StringVar(&args.agentType, "agent", "backend-developer", "Specify agent type (tester, debugger, researcher, backend-developer, frontend-developer, ux, ui, marketer, feedbackseeker, simplifier, documentationwriter). Agent-only execution is enforced.")
	flag.BoolVar(&args.listAgents, "list-agents", false, "List all available agent types")
	flag.BoolVar(&args.showProgress, "show-progress", false, "Show agent progress for current project")
	flag.BoolVar(&args.multiAgent, "multi-agent", false, "Run coordinated multi-agent session")
	flag.StringVar(&args.taskCommand, "task", "", "Task command (list, create, update, delete, stats)")
	flag.Parse()
	return args
}

func validateEnvironment() error {
	// Validate opencode CLI exists
	if _, err := exec.LookPath("opencode"); err != nil {
		return fmt.Errorf("opencode CLI not found. Please install it first")
	}

	// Validate required files exist
	requiredFiles := []string{"tasks.md", "progress.txt", "prompt.md"}
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("required file '%s' not found", file)
		}
	}

	return nil
}

// Status display mappings
var statusIcons = map[string]string{
	"pending":     "⏳",
	"in_progress": "🔄",
	"completed":   "✅",
	"cancelled":   "❌",
}

var statusNames = map[string]string{
	"pending":     "Pending",
	"in_progress": "In Progress",
	"completed":   "Completed",
	"cancelled":   "Cancelled",
}

func handleTaskCommand(command string) error {
	dbPath := "./giggum_tasks.db"
	taskManager, err := NewTaskManager(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create task manager: %v", err)
	}
	defer taskManager.Close()

	switch command {
	case "list":
		return displayTaskList(taskManager)
	case "stats":
		return displayTaskStats(taskManager)
	case "import":
		return importTasksFromMarkdown(taskManager)
	default:
		return fmt.Errorf("unknown task command: %s. Available: list, stats, import", command)
	}
}

func displayTaskList(taskManager *TaskManager) error {
	tasks, err := taskManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %v", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found. You can create tasks by editing tasks.md or using the import command.")
		return nil
	}

	// Group tasks by status
	statusGroups := make(map[string][]Task)
	for _, task := range tasks {
		statusGroups[task.Status] = append(statusGroups[task.Status], task)
	}

	// Display in order: pending, in_progress, completed, cancelled
	statusOrder := []string{"pending", "in_progress", "completed", "cancelled"}
	statusIcons := map[string]string{
		"pending":     "⏳",
		"in_progress": "🔄",
		"completed":   "✅",
		"cancelled":   "❌",
	}
	statusNames := map[string]string{
		"pending":     "Pending",
		"in_progress": "In Progress",
		"completed":   "Completed",
		"cancelled":   "Cancelled",
	}

	for _, status := range statusOrder {
		if tasks, exists := statusGroups[status]; exists && len(tasks) > 0 {
			fmt.Printf("\n%s %s (%d)\n", statusIcons[status], statusNames[status], len(tasks))
			fmt.Println(strings.Repeat("─", 50))

			for _, task := range tasks {
				priorityIcon := getPriorityIcon(task.Priority)
				fmt.Printf("%s [%d] %s\n", priorityIcon, task.ID, task.Title)

				if task.Description != "" && task.Description != task.Title {
					fmt.Printf("     %s\n", task.Description)
				}

				fmt.Printf("     Created: %s", task.CreatedAt.Format("2006-01-02 15:04"))
				if task.CompletedAt != nil {
					fmt.Printf(" | Completed: %s", task.CompletedAt.Format("2006-01-02 15:04"))
				}
				fmt.Println()

				if task.Tags != "" {
					fmt.Printf("     Tags: %s\n", task.Tags)
				}
				fmt.Println()
			}
		}
	}

	// Show summary
	stats, err := taskManager.GetTaskStats()
	if err == nil {
		fmt.Printf("\n📊 Summary: %d total tasks", stats["total"])
		if stats["pending"] > 0 {
			fmt.Printf(" | %d pending", stats["pending"])
		}
		if stats["in_progress"] > 0 {
			fmt.Printf(" | %d in progress", stats["in_progress"])
		}
		if stats["completed"] > 0 {
			fmt.Printf(" | %d completed", stats["completed"])
		}
		fmt.Println()
	}

	return nil
}

func getPriorityIcon(priority string) string {
	switch strings.ToLower(priority) {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func displayTaskStats(taskManager *TaskManager) error {
	stats, err := taskManager.GetTaskStats()
	if err != nil {
		return fmt.Errorf("failed to get task stats: %v", err)
	}

	fmt.Println("📊 Task Statistics")
	fmt.Println(strings.Repeat("═", 30))

	if stats["total"] == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	fmt.Printf("Total Tasks: %d\n\n", stats["total"])

	fmt.Println("By Status:")
	for status, count := range stats {
		if status != "total" {
			icon := statusIcons[status]
			name := statusNames[status]
			fmt.Printf("  %s %s: %d\n", icon, name, count)
		}
	}

	// Get all tasks to show priority breakdown
	tasks, err := taskManager.GetAllTasks()
	if err == nil {
		priorityCount := make(map[string]int)
		for _, task := range tasks {
			priorityCount[task.Priority]++
		}

		fmt.Println("\nBy Priority:")
		for priority, count := range priorityCount {
			icon := getPriorityIcon(priority)
			fmt.Printf("  %s %s: %d\n", icon, strings.Title(priority), count)
		}
	}

	return nil
}

func importTasksFromMarkdown(taskManager *TaskManager) error {
	// Check if tasks.md exists
	if _, err := os.Stat("tasks.md"); os.IsNotExist(err) {
		return fmt.Errorf("tasks.md file not found. Please create it first.")
	}

	content, err := os.ReadFile("tasks.md")
	if err != nil {
		return fmt.Errorf("failed to read tasks.md: %v", err)
	}

	err = taskManager.ImportTasksFromMarkdown(string(content))
	if err != nil {
		return fmt.Errorf("failed to import tasks: %v", err)
	}

	fmt.Println("✅ Tasks imported successfully from tasks.md")

	// Show count of imported tasks
	tasks, err := taskManager.GetAllTasks()
	if err == nil {
		fmt.Printf("📝 Total tasks in database: %d\n", len(tasks))
	}

	return nil
}

func runIterations(logger *Logger, config Config, iterations int, debug bool) {
	// Agent-only execution - validate agent type exists
	if config.AgentType == "" {
		config.AgentType = BackendDeveloper
	}

	for i := 1; i <= iterations; i++ {
		var err error
		fmt.Printf("Running iteration %d/%d with %s agent...\n", i, iterations, config.AgentType)

		// Execute using agent only - no traditional execution path
		// err := runAgentWithOutputCapture(logger, config.AgentType, config.PromptCommand, debug)
		if err != nil {
			if err.Error() == "COMPLETED" {
				// Send webhook notification for early completion
				if webhookErr := sendWebhookNotification(logger, config, i, true); webhookErr != nil {
					logger.Warn("Failed to send webhook notification: %v", webhookErr)
				}
				return // Exit early since tasks are complete
			}

			// Enhanced error handling for agent-only execution
			logger.Error("Agent execution failed in iteration %d: %v", i, err)
			// logger.Error("Agent type: %s, Task: %s", config.AgentType, config.PromptCommand)

			fmt.Fprintf(os.Stderr, "Error in iteration %d with %s agent: %v\n", i, config.AgentType, err)
			fmt.Fprintf(os.Stderr, "Note: This system uses agent-only execution - no fallback mode available.\n")

			if i == iterations {
				logger.Error("Final iteration failed, terminating")
				os.Exit(1)
			}
			logger.Info("Continuing to next iteration...")
			continue
		}
	}
}

func main() {
	args := parseFlags()

	// Handle task commands
	if args.taskCommand != "" {
		if err := handleTaskCommand(args.taskCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Error handling task command: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle special commands that don't need full initialization
	if args.listAgents {
		listAvailableAgents()
		return
	}

	if args.showProgress {
		showAgentProgress()
		return
	}

	if args.help {
		showHelp()
		return
	}

	if args.restore {
		logger, err := NewLogger("INFO", args.verbose, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Close()

		if err := restoreProgress(logger); err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring progress: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Debug implies verbose
	if args.debug {
		args.verbose = true
	}

	// Validate environment
	if err := validateEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse iteration count from argument if provided
	if flag.NArg() > 0 {
		var parsedIterations int
		_, err := fmt.Sscanf(flag.Arg(0), "%d", &parsedIterations)
		if err == nil {
			args.iterations = parsedIterations
		}
	}

	// Create logger
	logger, err := NewLogger("INFO", args.verbose, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Load configuration
	config, err := loadConfig(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line agent settings
	// Agent-only execution enforced - no traditional execution mode available
	config.UseAgents = true
	if args.agentType != "" {
		config.AgentType = AgentType(args.agentType)
	} else if config.AgentType == "" {
		// Ensure we have a default agent type if none was specified in config or CLI
		config.AgentType = BackendDeveloper
	}

	logger.Info("CLI: Agent-only execution enforced with %s agent", config.AgentType)

	// Handle multi-agent mode
	if args.multiAgent {
		if err := runMultiAgentSession(logger, config, args.iterations, args.debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error in multi-agent session: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Validate agent type - always required since we use agent-only execution
	if config.AgentType != "" {
		if _, err := GetAgentPrompt(config.AgentType); err != nil {
			logger.Error("Invalid agent type '%s': %v", config.AgentType, err)
			fmt.Fprintf(os.Stderr, "Error: Invalid agent type '%s'. This system requires agent-only execution.\n", config.AgentType)
			fmt.Fprintf(os.Stderr, "Use -list-agents to see available agent types.\n")
			os.Exit(1)
		}
		logger.Info("Agent type '%s' validated successfully", config.AgentType)
	}

	// Validate feedback loops
	if err := validateFeedbackLoops(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error validating feedback loops: %v\n", err)
		os.Exit(1)
	}

	// Backup progress if requested
	if args.backup {
		if err := backupProgress(logger); err != nil {
			logger.Warn("Failed to backup progress: %v", err)
		}
	}

	// Run iterations
	runIterations(logger, config, args.iterations, args.debug)

	// Send webhook notification after completing all iterations
	if err := sendWebhookNotification(logger, config, args.iterations, true); err != nil {
		logger.Warn("Failed to send webhook notification: %v", err)
	}
}
