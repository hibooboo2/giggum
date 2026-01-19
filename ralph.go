package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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
	flag.StringVar(&args.taskCommand, "task", "", "Task command (list, create, add, remove, stats, import)")
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

var taskManager *TaskManager

func handleTaskCommand(command string) error {
	switch command {
	case "list":
		return displayTaskList(taskManager)
	case "create":
		return createTaskInteractively(taskManager)
	case "add":
		return addTaskFromRawString(taskManager)
	case "remove":
		return removeTaskInteractively(taskManager)
	case "stats":
		return displayTaskStats(taskManager)
	default:
		return fmt.Errorf("unknown task command: %s. Available: list, create, add, remove, stats, import", command)
	}
}

func addTaskFromRawString(taskManager *TaskManager) error {
	// Get the raw task string from command line arguments
	if flag.NArg() < 1 {
		return fmt.Errorf("task add requires a task string. Usage: giggum -task add \"your task description\"")
	}

	rawTask := strings.Join(flag.Args(), " ")
	if strings.TrimSpace(rawTask) == "" {
		return fmt.Errorf("task string cannot be empty")
	}

	fmt.Printf("🤖 Analyzing task with researcher agent...\n")
	fmt.Printf("📝 Raw input: %s\n\n", rawTask)

	// Create a logger for the researcher agent
	logger, err := NewLogger("INFO", false, "")
	if err != nil {
		return fmt.Errorf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Prepare researcher prompt
	researcherPrompt := fmt.Sprintf(`As a thorough Researcher with expertise in technical investigation, requirement analysis, and solution exploration, analyze this task and provide structured metadata:

TASK TO ANALYZE: %s

Please analyze this task and provide a structured response with the following format:

TITLE: [Clear, concise title for the task]
DESCRIPTION: [Detailed description of what needs to be done]
PRIORITY: [high/medium/low based on urgency and importance]
TAGS: [comma-separated relevant tags like "frontend", "backend", "database", "ui", "api", etc.]
METADATA: [JSON object with additional context like complexity, estimated_time, dependencies, etc.]

Focus on:
1. Breaking down the task into clear requirements
2. Identifying the technical domain and scope
3. Assessing priority based on business impact and urgency
4. Extracting relevant tags for categorization
5. Providing metadata that helps with task management

Provide only the structured response, no additional explanation.`, rawTask)

	fmt.Printf("🔍 Researching task requirements...\n")
	err = runAgent(logger, Researcher, researcherPrompt, false)
	if err != nil {
		fmt.Printf("⚠️  Researcher agent analysis failed, proceeding with basic analysis: %v\n", err)
		// Continue with basic analysis even if researcher fails
	}

	// Note: In a real implementation, we would capture the researcher's output
	// For now, we'll create a basic task with the raw string as both title and description
	// and let the user refine it later

	// Extract basic info from the raw task for now
	lines := strings.Split(rawTask, ".")
	title := lines[0]
	if len(title) > 100 {
		title = title[:97] + "..."
	}
	description := rawTask

	// Basic heuristics for priority
	priority := "medium"
	lowerTask := strings.ToLower(rawTask)
	if strings.Contains(lowerTask, "urgent") || strings.Contains(lowerTask, "critical") || strings.Contains(lowerTask, "asap") {
		priority = "high"
	} else if strings.Contains(lowerTask, "later") || strings.Contains(lowerTask, "when") || strings.Contains(lowerTask, "eventually") {
		priority = "low"
	}

	// Basic heuristics for tags
	tags := ""
	if strings.Contains(lowerTask, "database") || strings.Contains(lowerTask, "db") || strings.Contains(lowerTask, "sql") {
		tags = "database"
	} else if strings.Contains(lowerTask, "frontend") || strings.Contains(lowerTask, "ui") || strings.Contains(lowerTask, "interface") {
		tags = "frontend"
	} else if strings.Contains(lowerTask, "backend") || strings.Contains(lowerTask, "api") || strings.Contains(lowerTask, "server") {
		tags = "backend"
	} else if strings.Contains(lowerTask, "test") || strings.Contains(lowerTask, "testing") {
		tags = "testing"
	}

	// Create the task
	task, err := taskManager.CreateTask(title, description, priority)
	if err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}

	// Update tags and metadata if provided
	metadata := fmt.Sprintf(`{"source": "task_add_command", "raw_input": "%s", "analysis_date": "%s"}`,
		strings.ReplaceAll(rawTask, "\"", "'"),
		time.Now().Format("2006-01-02T15:04:05Z"))

	if tags != "" || metadata != "" {
		if err := taskManager.UpdateTaskMetadata(task.ID, tags, metadata); err != nil {
			fmt.Printf("Warning: failed to save tags/metadata: %v\n", err)
		} else {
			// Refresh the task to get updated values
			task, _ = taskManager.GetTask(task.ID)
		}
	}

	// Display the created task
	fmt.Printf("\n✅ Task created successfully!\n")
	fmt.Printf("📋 ID: %d\n", task.ID)
	fmt.Printf("📝 Title: %s\n", task.Title)
	if task.Description != "" {
		fmt.Printf("📄 Description: %s\n", task.Description)
	}
	fmt.Printf("🎯 Priority: %s\n", task.Priority)
	fmt.Printf("📅 Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("📊 Status: %s\n", task.Status)
	if task.Tags != "" {
		fmt.Printf("🏷️  Tags: %s\n", task.Tags)
	}

	return nil
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

func getPriorityNum(priority string) int {
	switch strings.ToLower(priority) {
	case "high":
		return 10
	case "medium":
		return 5
	case "low":
		return 1
	default:
		return 0
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

func createTaskInteractively(taskManager *TaskManager) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("📝 Create New Task")
	fmt.Println("=================")

	// Get task title
	fmt.Print("Task title: ")
	title, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read title: %v", err)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("task title cannot be empty")
	}

	// Get task description
	fmt.Print("Task description (optional): ")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	// Get priority
	fmt.Print("Priority (high/medium/low) [medium]: ")
	priorityInput, _ := reader.ReadString('\n')
	priority := strings.ToLower(strings.TrimSpace(priorityInput))
	if priority == "" {
		priority = "medium"
	}
	if priority != "high" && priority != "medium" && priority != "low" {
		fmt.Printf("Invalid priority '%s', using 'medium'\n", priority)
		priority = "medium"
	}

	// Get tags
	fmt.Print("Tags (comma-separated, optional): ")
	tagsInput, _ := reader.ReadString('\n')
	tags := strings.TrimSpace(tagsInput)

	// Get metadata
	fmt.Print("Additional metadata (optional): ")
	metadataInput, _ := reader.ReadString('\n')
	metadata := strings.TrimSpace(metadataInput)

	// Create the task
	task, err := taskManager.CreateTask(title, description, priority)
	if err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}

	// Update tags and metadata if provided
	if tags != "" || metadata != "" {
		if err := taskManager.UpdateTaskMetadata(task.ID, tags, metadata); err != nil {
			fmt.Printf("Warning: failed to save tags/metadata: %v\n", err)
		} else {
			// Refresh the task to get updated values
			task, _ = taskManager.GetTask(task.ID)
		}
	}

	// Display the created task
	fmt.Printf("\n✅ Task created successfully!\n")
	fmt.Printf("📋 ID: %d\n", task.ID)
	fmt.Printf("📝 Title: %s\n", task.Title)
	if task.Description != "" {
		fmt.Printf("📄 Description: %s\n", task.Description)
	}
	fmt.Printf("🎯 Priority: %s\n", task.Priority)
	fmt.Printf("📅 Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("📊 Status: %s\n", task.Status)

	return nil
}

func removeTaskInteractively(taskManager *TaskManager) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🗑️  Remove Task")
	fmt.Println("================")

	// Show current tasks first
	tasks, err := taskManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %v", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found to remove.")
		return nil
	}

	// Display tasks for selection
	fmt.Println("\nCurrent tasks:")
	fmt.Println(strings.Repeat("─", 50))
	for _, task := range tasks {
		priorityIcon := getPriorityIcon(task.Priority)
		statusIcon := statusIcons[task.Status]
		fmt.Printf("%s %s [%d] %s\n", priorityIcon, statusIcon, task.ID, task.Title)
		if task.Description != "" && task.Description != task.Title {
			fmt.Printf("     %s\n", task.Description)
		}
		fmt.Printf("     Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Println()
	}

	// Get task ID to remove
	for {
		fmt.Print("Enter task ID to remove (or 'q' to cancel): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %v", err)
		}
		input = strings.TrimSpace(input)

		if input == "q" || input == "quit" || input == "exit" {
			fmt.Println("Task removal cancelled.")
			return nil
		}

		var taskID int64
		if _, err := fmt.Sscanf(input, "%d", &taskID); err != nil {
			fmt.Printf("Invalid task ID. Please enter a number or 'q' to cancel.\n")
			continue
		}

		// Verify task exists
		task, err := taskManager.GetTask(taskID)
		if err != nil {
			fmt.Printf("Task with ID %d not found. Please try again.\n", taskID)
			continue
		}

		// Show task details and confirm
		fmt.Printf("\nTask to remove:\n")
		priorityIcon := getPriorityIcon(task.Priority)
		statusIcon := statusIcons[task.Status]
		fmt.Printf("%s %s [%d] %s\n", priorityIcon, statusIcon, task.ID, task.Title)
		if task.Description != "" && task.Description != task.Title {
			fmt.Printf("Description: %s\n", task.Description)
		}
		fmt.Printf("Status: %s | Priority: %s\n", task.Status, task.Priority)
		fmt.Printf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))

		// Confirmation
		fmt.Printf("\nAre you sure you want to remove this task? (y/N): ")
		confirm, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %v", err)
		}
		confirm = strings.ToLower(strings.TrimSpace(confirm))

		if confirm == "y" || confirm == "yes" {
			// Delete the task
			if err := taskManager.DeleteTask(taskID); err != nil {
				return fmt.Errorf("failed to remove task: %v", err)
			}
			fmt.Printf("✅ Task [%d] removed successfully.\n", taskID)
			return nil
		} else {
			fmt.Printf("Task removal cancelled.\n")
			return nil
		}
	}
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

	dbPath := "./giggum_tasks.db"
	var err error
	taskManager, err = NewTaskManager(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create task manager: %v", err)
	}
	defer taskManager.Close()

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
