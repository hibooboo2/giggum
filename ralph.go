package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	verbose bool
}

// Config holds the configuration for the Ralph Wiggum loop
type Config struct {
	PromptCommand string    `json:"prompt_command"`
	WebhookURL    string    `json:"webhook_url"`
	ParseReply    bool      `json:"wait_for_reply"`
	ReplyPrompt   string    `json:"reply_prompt"`
	AddToTasks    bool      `json:"add_to_tasks"`
	AgentType     AgentType `json:"agent_type,omitempty"`
	UseAgents     bool      `json:"use_agents"`
	// Future configuration options can be added here
}

// DefaultPromptCommand is the default prompt command if not specified in config
const DefaultPromptCommand = "@tasks.md @progress.txt @prompt.md Follow the instrunctions in prompt.md"

// NewLogger creates a new logger instance using slog
func NewLogger(logLevel string, verbose bool, logFilePath string) (*Logger, error) {
	// Parse log level
	var level slog.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if logFilePath != "" {
		// Create file handler
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file '%s': %v", logFilePath, err)
		}
		handler = slog.NewTextHandler(file, opts)
	} else {
		// Use stdout handler
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	return &Logger{
		Logger:  logger,
		verbose: verbose,
	}, nil
}

// Close closes any resources (placeholder for compatibility)
func (l *Logger) Close() error {
	// slog handles resources automatically, no explicit close needed
	return nil
}

// Debug logs a debug message with custom verbose handling
func (l *Logger) Debug(msg string, args ...any) {
	if l.verbose {
		l.Logger.Debug(msg, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// createAgentPrompt creates a prompt for a specific agent type
func createAgentPrompt(agentType AgentType, task string) (string, error) {
	prompt, err := GetAgentPrompt(agentType)
	if err != nil {
		return "", fmt.Errorf("failed to get agent prompt: %v", err)
	}

	return fmt.Sprintf(prompt.TaskPrompt, task), nil
}

// runAgent executes a task using a specific agent type
func runAgent(logger *Logger, agentType AgentType, task string, debug bool) error {
	// Get the current working directory for project path
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	// Initialize database manager
	dbPath := GetProjectDBPath(projectPath)
	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		logger.Warn("Failed to initialize database: %v", err)
		// Continue without database if it fails
	} else {
		defer dbManager.Close()
	}

	// Create agent session
	var sessionID int64
	if dbManager != nil {
		sessionID, err = dbManager.CreateAgentSession(agentType, projectPath)
		if err != nil {
			logger.Warn("Failed to create agent session: %v", err)
		}
	}

	// Create agent-specific prompt
	agentPrompt, err := createAgentPrompt(agentType, task)
	if err != nil {
		return fmt.Errorf("failed to create agent prompt: %v", err)
	}

	logger.Info("Running %s agent for task: %s", agentType, task)

	// Build the opencode command
	args := []string{"run", "--model", "opencode/big-pickle"}
	if debug {
		args = append(args, "--print-logs")
	}

	// Combine the system prompt with the task prompt
	agentPromptInfo, _ := GetAgentPrompt(agentType)
	fullPrompt := fmt.Sprintf("%s\n\n%s", agentPromptInfo.SystemPrompt, agentPrompt)
	args = append(args, fullPrompt)

	cmd := exec.Command("opencode", args...)
	cmd.Env = append(os.Environ(), "OPENAI_BASE_URL=http://100.83.162.29:1234")

	// Set up output capture
	var outputBuffer bytes.Buffer
	var writer io.Writer = &outputBuffer

	writer = io.MultiWriter(&outputBuffer, os.Stdout)

	cmd.Stdout = writer
	cmd.Stderr = writer

	// Add input to database
	if dbManager != nil && sessionID > 0 {
		dbManager.AddSessionInput(sessionID, fullPrompt)
	}

	// Run the command
	err = cmd.Run()
	output := outputBuffer.String()

	// Add output to database
	if dbManager != nil && sessionID > 0 {
		dbManager.AddSessionOutput(sessionID, output)
	}

	if err != nil {
		return fmt.Errorf("agent execution failed: %v", err)
	}

	// Close the session
	if dbManager != nil && sessionID > 0 {
		dbManager.EndAgentSession(sessionID)
	}

	return nil
}

// loadConfig loads configuration from config.json file, returns default config if file doesn't exist
func loadConfig(logger *Logger) (*Config, error) {
	configFile := "config.json"

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		logger.Debug("Config file '%s' not found, using default configuration", configFile)
		return &Config{
			PromptCommand: DefaultPromptCommand,
			ParseReply:    false,
			ReplyPrompt:   "Enter your response (or press Enter to continue): ",
			AddToTasks:    true,
			UseAgents:     false,
			AgentType:     "",
		}, nil
	}

	// Read config file
	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %v", configFile, err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %v", configFile, err)
	}

	// Use default prompt command if not specified
	if config.PromptCommand == "" {
		config.PromptCommand = DefaultPromptCommand
	}

	// Set defaults for reply functionality
	if config.ReplyPrompt == "" {
		config.ReplyPrompt = "Enter your response (or press Enter to continue): "
	}

	// Set default for agent functionality
	if config.AgentType == "" && config.UseAgents {
		config.AgentType = BackendDeveloper // Default to backend developer
	}

	logger.Debug("Loaded configuration from '%s'", configFile)

	return &config, nil
}

// processWebhookResponse processes a webhook response by sending it to opencode to create a task list
func processWebhookResponse(logger *Logger, config Config, response string) error {
	if strings.TrimSpace(response) == "" {
		logger.Debug("Empty webhook response, skipping processing")
		return nil
	}

	logger.Info("Processing webhook response: %s", response)

	// If configured to add to tasks.md
	if config.AddToTasks {
		if err := addToTasks(logger, response); err != nil {
			logger.Warn("Failed to add webhook response to tasks.md: %v", err)
			return err
		}
	}

	return nil
}

// addToTasks adds a response to the tasks.md file using opencode for processing
func addToTasks(logger *Logger, response string) error {
	// If response is empty, skip processing
	if strings.TrimSpace(response) == "" {
		logger.Debug("Empty response, skipping task addition")
		return nil
	}

	// Use opencode to process the webhook response into a formatted task list
	logger.Info("Processing webhook response with opencode to create task list...")

	// Create opencode prompt for task processing
	opencodePrompt := fmt.Sprintf(`Process the following webhook response and extract/create a proper task list. 
The response may contain new tasks, feedback, or requirements. Convert them into a clean, organized task list format.

Webhook Response:
%s

Requirements:
1. Extract individual tasks from the response
2. Format each task as a markdown list item starting with "- "
3. Make tasks specific and actionable
4. Remove any duplicate or irrelevant content
5. If no clear tasks are found, respond with "No tasks found"
6. Output ONLY the task list, no explanations

Example output format:
- Implement user authentication system
- Add API endpoint for user registration
- Create login form component
`, response)

	// Build the opencode command
	args := []string{"run", "--model", "opencode/big-pickle"}
	cmd := exec.Command("opencode", args...)
	cmd.Env = append(os.Environ(), "OPENAI_BASE_URL=http://100.83.162.29:1234")

	// Provide the prompt via stdin
	cmd.Stdin = strings.NewReader(opencodePrompt)

	// Capture the output
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	// Run the command
	err := cmd.Run()
	if err != nil {
		logger.Warn("Failed to process webhook response with opencode: %v", err)
		logger.Warn("Using fallback: adding raw response as task")
		return addRawTask(logger, response)
	}

	// Get the processed task list
	processedTasks := strings.TrimSpace(output.String())

	// Check if opencode found any tasks
	if processedTasks == "" || strings.Contains(strings.ToLower(processedTasks), "no tasks found") {
		logger.Info("No tasks found in webhook response")
		return nil
	}

	// Read current tasks.md content
	content, err := os.ReadFile("tasks.md")
	if err != nil {
		return fmt.Errorf("failed to read tasks.md: %v", err)
	}

	// Create new content with the processed tasks added
	newContent := string(content)
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	// Add a separator and the new tasks
	newContent += "\n# Tasks from Webhook Response\n"
	newContent += processedTasks + "\n"

	// Write back to tasks.md
	if err := os.WriteFile("tasks.md", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write to tasks.md: %v", err)
	}

	logger.Info("Added processed tasks to tasks.md from webhook response")
	return nil
}

// addRawTask is a fallback function that adds the raw response as a single task
func addRawTask(logger *Logger, response string) error {
	// Read current tasks.md content
	content, err := os.ReadFile("tasks.md")
	if err != nil {
		return fmt.Errorf("failed to read tasks.md: %v", err)
	}

	// Create new content with the response added
	newContent := string(content)
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += fmt.Sprintf("- %s\n", response)

	// Write back to tasks.md
	if err := os.WriteFile("tasks.md", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write to tasks.md: %v", err)
	}

	logger.Info("Added raw response to tasks.md: %s", response)
	return nil
}

// sendWebhookNotification sends a notification to the configured webhook URL
func sendWebhookNotification(logger *Logger, config Config, iterations int, completed bool) error {
	if config.WebhookURL == "" {
		logger.Debug("No webhook URL configured, skipping notification")
		return nil
	}

	// Prepare webhook payload
	payload := map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"iterations":     iterations,
		"completed":      completed,
		"message":        "Ralph Wiggum execution completed",
		"reply_endpoint": "http://localhost:8080/reply",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	// Send HTTP POST request
	resp, err := http.Post(config.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	logger.Info("Webhook notification sent successfully to %s", config.WebhookURL)

	// If ParseReply is enabled, wait for and process the webhook response
	if config.ParseReply {
		// Read the immediate response from the webhook
		respdata, _ := io.ReadAll(resp.Body)
		response := strings.TrimSpace(string(respdata))

		if response != "" {
			if err := processWebhookResponse(logger, config, response); err != nil {
				logger.Warn("Failed to process webhook response: %v", err)
			}
		} else {
			logger.Debug("No immediate response received from webhook")
		}
	}
	return nil
}

func main() {
	// Define simple command line flags
	var help bool
	var iterations int
	var verbose bool
	var debug bool
	var backup bool
	var restore bool
	var agentType string
	var useAgents bool
	var listAgents bool
	var showProgress bool
	var multiAgent bool

	flag.BoolVar(&help, "h", false, "Show help message")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.IntVar(&iterations, "n", 10, "Number of iterations to run")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&debug, "debug", false, "Enable debug output (implies -v)")
	flag.BoolVar(&backup, "backup", false, "Backup progress before running")
	flag.BoolVar(&restore, "restore", false, "Restore progress from latest backup and exit")
	flag.StringVar(&agentType, "agent", "", "Specify agent type (tester, debugger, researcher, backend-developer, frontend-developer, ux, ui, marketer, feedbackseeker, simplifier, documentationwriter)")
	flag.BoolVar(&useAgents, "use-agents", false, "Enable multi-agent mode")
	flag.BoolVar(&listAgents, "list-agents", false, "List all available agent types")
	flag.BoolVar(&showProgress, "show-progress", false, "Show agent progress for current project")
	flag.BoolVar(&multiAgent, "multi-agent", false, "Run coordinated multi-agent session")
	flag.Parse()

	// List agents if requested
	if listAgents {
		listAvailableAgents()
		return
	}

	// Show progress if requested
	if showProgress {
		showAgentProgress()
		return
	}

	// Show help if requested
	if help {
		showHelp()
		return
	}

	// Handle restore flag
	if restore {
		logger, err := NewLogger("INFO", verbose, "")
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
	if debug {
		verbose = true
	}

	// Validate opencode CLI exists
	if _, err := exec.LookPath("opencode"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: opencode CLI not found. Please install it first.\n")
		os.Exit(1)
	}

	// Validate required files exist
	requiredFiles := []string{"tasks.md", "progress.txt", "prompt.md"}
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Required file '%s' not found.\n", file)
			os.Exit(1)
		}
	}

	// Parse iteration count from argument if provided
	if flag.NArg() > 0 {
		var parsedIterations int
		_, err := fmt.Sscanf(flag.Arg(0), "%d", &parsedIterations)
		if err == nil {
			iterations = parsedIterations
		}
	}

	// Create logger for configuration loading
	logger, err := NewLogger("INFO", verbose, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Load configuration using the loadConfig function
	config, err := loadConfig(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line agent settings
	if useAgents {
		config.UseAgents = true
	}
	if agentType != "" {
		config.AgentType = AgentType(agentType)
		config.UseAgents = true
	}

	// Handle multi-agent mode
	if multiAgent {
		if err := runMultiAgentSession(logger, iterations, debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error in multi-agent session: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Validate agent type if specified
	if config.UseAgents && config.AgentType != "" {
		if _, err := GetAgentPrompt(config.AgentType); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid agent type '%s'. Use -list-agents to see available types.\n", config.AgentType)
			os.Exit(1)
		}
	}

	// Validate feedback loops and required files
	if err := validateFeedbackLoops(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error validating feedback loops: %v\n", err)
		os.Exit(1)
	}

	// Backup progress if requested
	if backup {
		if err := backupProgress(logger); err != nil {
			logger.Warn("Failed to backup progress: %v", err)
		}
	}

	// Run the autonomous coding loop
	for i := 1; i <= iterations; i++ {
		fmt.Printf("Running iteration %d/%d...\n", i, iterations)

		var err error
		output := ""

		if config.UseAgents && config.AgentType != "" {
			// Run using agent mode
			fmt.Printf("Using %s agent...\n", config.AgentType)
			err = runAgent(logger, config.AgentType, config.PromptCommand, debug)
		} else {
			// Run using traditional mode
			args := []string{"run", "--model", "opencode/big-pickle"}
			if debug {
				args = append(args, "--print-logs")
			}
			args = append(args, config.PromptCommand)
			cmd := exec.Command("opencode", args...)
			cmd.Env = append(os.Environ(), "OPENAI_BASE_URL=http://100.83.162.29:1234")

			// Set up output capture for tee reader functionality
			var outputBuffer bytes.Buffer
			var writer io.Writer = &outputBuffer

			if verbose {
				// Use MultiWriter to capture output while displaying it
				writer = io.MultiWriter(&outputBuffer, os.Stdout)
			}

			cmd.Stdout = writer
			cmd.Stderr = writer

			// Run the command
			err = cmd.Run()
			output = outputBuffer.String()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in iteration %d: %v\n", i, err)
			if i == iterations {
				os.Exit(1)
			}
			continue
		}

		// Check for completion from captured output
		if strings.Contains(output, "✅ Complete ✅") {
			fmt.Println("✅ All tasks completed!")
			// Send webhook notification for early completion
			if err := sendWebhookNotification(logger, *config, i, false); err != nil {
				logger.Warn("Failed to send webhook notification: %v", err)
			}
		}
	}

	// Send webhook notification after completing all iterations
	if err := sendWebhookNotification(logger, *config, -1, false); err != nil {
		logger.Warn("Failed to send webhook notification: %v", err)
	}
}

func validateFeedbackLoops(logger *Logger) error {
	// Check if opencode CLI is available
	if _, err := exec.LookPath("opencode"); err != nil {
		return fmt.Errorf("opencode CLI not found. Please install the opencode CLI and ensure it's in your PATH: %v", err)
	}

	// Validate required files exist and provide helpful messages
	requiredFiles := map[string]string{
		"tasks.md":     "Task definitions and priorities",
		"progress.txt": "Progress tracking file",
		"prompt.md":    "AI execution prompt",
	}

	for file, description := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("required file '%s' not found. This file should contain: %s\nTo create it, you can run: touch %s", file, description, file)
		}

		// Check if file is readable and not empty
		info, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("cannot access file '%s': %v", file, err)
		}

		if info.Size() == 0 {
			logger.Warn("File '%s' is empty. Consider adding content to ensure proper operation.", file)
		}

		logger.Debug("File '%s' exists and is accessible (%d bytes)", file, info.Size())
	}

	logger.Debug("Feedback loop validation passed")
	logger.Debug("opencode CLI is available")
	logger.Debug("All required files exist and are accessible")

	return nil
}

// backupProgress creates a timestamped backup of progress.txt
func backupProgress(logger *Logger) error {
	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("progress_%s.txt", timestamp))

	// Read the original file
	content, err := os.ReadFile("progress.txt")
	if err != nil {
		return fmt.Errorf("failed to read progress.txt: %v", err)
	}

	// Write to backup file
	if err := os.WriteFile(backupFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %v", err)
	}

	logger.Debug("Created backup: %s", backupFile)

	return nil
}

// restoreProgress restores progress.txt from the latest backup
func restoreProgress(logger *Logger) error {
	backupDir := "backups"
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup directory '%s' does not exist", backupDir)
	}

	// Find the most recent backup file
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if strings.HasPrefix(name, "progress_") && strings.HasSuffix(name, ".txt") {
			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = filepath.Join(backupDir, name)
			}
		}
	}

	if latestFile == "" {
		return fmt.Errorf("no backup files found in %s", backupDir)
	}

	// Read backup file
	content, err := os.ReadFile(latestFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}

	// Write to progress.txt
	if err := os.WriteFile("progress.txt", content, 0644); err != nil {
		return fmt.Errorf("failed to restore progress.txt: %v", err)
	}

	logger.Info("Restored progress.txt from: %s", latestFile)

	return nil
}

func listAvailableAgents() {
	fmt.Printf("Available Agent Types:\n\n")

	prompts := GetAgentPrompts()
	for _, agentType := range ListAgentTypes() {
		prompt := prompts[agentType]
		fmt.Printf("  %-20s - %s\n", prompt.Name, prompt.Description)
	}

	fmt.Printf("\nUsage: ralph -agent <agent-type>\n")
}

func showAgentProgress() {
	// Get current working directory for project path
	projectPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize database manager
	dbPath := GetProjectDBPath(projectPath)
	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer dbManager.Close()

	// Get all agent progress
	progressMap, err := dbManager.GetAllAgentProgress(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting agent progress: %v\n", err)
		os.Exit(1)
	}

	// Get project statistics
	stats, err := dbManager.GetProjectStats(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting project stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Agent Progress for Project: %s\n\n", projectPath)

	// Display session statistics
	if sessions, ok := stats["sessions"].(map[string]int); ok {
		fmt.Printf("Session Statistics:\n")
		for agentType, count := range sessions {
			fmt.Printf("  %-20s: %d sessions\n", agentType, count)
		}
		fmt.Println()
	}

	// Display progress by agent type
	if len(progressMap) == 0 {
		fmt.Printf("No progress found for any agents.\n")
		return
	}

	for _, agentType := range ListAgentTypes() {
		agentProgress, exists := progressMap[agentType]
		if !exists || len(agentProgress) == 0 {
			continue
		}

		prompt, _ := GetAgentPrompt(agentType)
		fmt.Printf("=== %s (%s) ===\n", prompt.Name, agentType)

		for _, p := range agentProgress {
			statusIcon := "⏳"
			switch p.Status {
			case "completed":
				statusIcon = "✅"
			case "in_progress":
				statusIcon = "🔄"
			case "failed":
				statusIcon = "❌"
			}

			fmt.Printf("  %s %s\n", statusIcon, p.Task)
			if p.Progress != "" {
				fmt.Printf("     %s\n", p.Progress)
			}
			fmt.Printf("     %s\n", p.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Println()
		}
		fmt.Println()
	}
}

// runMultiAgentSession coordinates multiple agents to work on tasks
func runMultiAgentSession(logger *Logger, iterations int, debug bool) error {
	// Read tasks.md to get the list of tasks
	tasksContent, err := os.ReadFile("tasks.md")
	if err != nil {
		return fmt.Errorf("failed to read tasks.md: %v", err)
	}

	// Parse tasks from the file
	tasks := parseTasks(string(tasksContent))
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found in tasks.md")
	}

	// Define agent priorities for different task types
	agentPriority := []AgentType{
		BackendDeveloper,
		FrontendDeveloper,
		Tester,
		Debugger,
		Researcher,
		UX,
		UI,
		DocumentationWriter,
		FeedbackSeeker,
		Marketer,
		Simplifier,
	}

	fmt.Printf("Starting multi-agent session with %d tasks\n", len(tasks))

	for i := 0; i < iterations && i < len(tasks); i++ {
		task := tasks[i]

		// Determine the best agent for this task
		agentType := selectBestAgentForTask(task, agentPriority)

		fmt.Printf("Iteration %d/%d: Using %s agent for task: %s\n",
			i+1, iterations, agentType, task)

		// Run the agent
		err := runAgent(logger, agentType, task, debug)
		if err != nil {
			logger.Warn("Agent %s failed on task '%s': %v", agentType, task, err)
			// Try with a different agent as fallback
			if agentType != Debugger {
				fmt.Printf("Retrying with Debugger agent...\n")
				err := runAgent(logger, Debugger, task, debug)
				if err != nil {
					logger.Error("Debugger also failed on task '%s': %v", task, err)
				}
			}
		}

		// Add separator between tasks
		if i < iterations-1 && i < len(tasks)-1 {
			fmt.Printf("\n%s\n\n", strings.Repeat("-", 50))
		}
	}

	return nil
}

// parseTasks extracts tasks from tasks.md content
func parseTasks(content string) []string {
	var tasks []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			task := strings.TrimPrefix(line, "- ")
			task = strings.TrimSpace(task)
			if task != "" {
				tasks = append(tasks, task)
			}
		}
	}

	return tasks
}

// selectBestAgentForTask determines which agent type is best suited for a task
func selectBestAgentForTask(task string, priority []AgentType) AgentType {
	taskLower := strings.ToLower(task)

	// Simple keyword-based agent selection
	if strings.Contains(taskLower, "test") || strings.Contains(taskLower, "bug") || strings.Contains(taskLower, "quality") {
		return Tester
	}
	if strings.Contains(taskLower, "debug") || strings.Contains(taskLower, "fix") || strings.Contains(taskLower, "error") {
		return Debugger
	}
	if strings.Contains(taskLower, "research") || strings.Contains(taskLower, "investigate") || strings.Contains(taskLower, "analyze") {
		return Researcher
	}
	if strings.Contains(taskLower, "backend") || strings.Contains(taskLower, "api") || strings.Contains(taskLower, "database") {
		return BackendDeveloper
	}
	if strings.Contains(taskLower, "frontend") || strings.Contains(taskLower, "ui") || strings.Contains(taskLower, "interface") {
		return FrontendDeveloper
	}
	if strings.Contains(taskLower, "ux") || strings.Contains(taskLower, "user experience") {
		return UX
	}
	if strings.Contains(taskLower, "design") || strings.Contains(taskLower, "visual") {
		return UI
	}
	if strings.Contains(taskLower, "document") || strings.Contains(taskLower, "readme") || strings.Contains(taskLower, "wiki") {
		return DocumentationWriter
	}
	if strings.Contains(taskLower, "market") || strings.Contains(taskLower, "promote") || strings.Contains(taskLower, "content") {
		return Marketer
	}
	if strings.Contains(taskLower, "feedback") || strings.Contains(taskLower, "review") {
		return FeedbackSeeker
	}
	if strings.Contains(taskLower, "simplify") || strings.Contains(taskLower, "explain") {
		return Simplifier
	}

	// Default to the first agent in priority list
	return priority[0]
}

func showHelp() {
	fmt.Printf(`Ralph Wiggum - Simple Autonomous AI Coding Loop

 USAGE:
    ralph [OPTIONS] [ITERATIONS]

 OPTIONS:
    -h, --help      Show this help message
    -v              Enable verbose output (shows colorized opencode output)
    -debug          Enable debug output (implies -v, adds --print-logs to opencode)
    -n N            Number of iterations to run (default: 10)
    -backup         Backup progress.txt before running
    -restore        Restore progress.txt from latest backup and exit
    -agent TYPE     Use specific agent type (see -list-agents for available types)
    -use-agents     Enable multi-agent mode
    -multi-agent    Run coordinated multi-agent session
    -list-agents    List all available agent types and their descriptions
    -show-progress  Show agent progress for current project

 EXAMPLES:
    ralph           # Run 10 iterations with default behavior
    ralph 5         # Run 5 iterations  
    ralph -n 20 -v  # Run 20 iterations with verbose output
    ralph -debug    # Run with debug output (shows opencode logs)
    ralph -agent tester  # Run using the tester agent
    ralph -multi-agent   # Run coordinated multi-agent session
    ralph -show-progress # Show agent progress
    ralph -list-agents  # Show all available agent types
    ralph -h        # Show this help

 REQUIRED FILES:
    - tasks.md      Task definitions
    - progress.txt  Progress tracking
    - prompt.md     AI execution prompt

 OPTIONAL FILES:
    - config.json   Custom prompt command and agent settings

 AGENT TYPES:
    Use -list-agents to see all available agent types and their descriptions.

`)
}
