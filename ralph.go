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
	PromptCommand string `json:"prompt_command"`
	WebhookURL    string `json:"webhook_url"`
	ParseReply    bool   `json:"wait_for_reply"`
	ReplyPrompt   string `json:"reply_prompt"`
	AddToTasks    bool   `json:"add_to_tasks"`
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

	flag.BoolVar(&help, "h", false, "Show help message")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.IntVar(&iterations, "n", 10, "Number of iterations to run")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&debug, "debug", false, "Enable debug output (implies -v)")
	flag.BoolVar(&backup, "backup", false, "Backup progress before running")
	flag.BoolVar(&restore, "restore", false, "Restore progress from latest backup and exit")
	flag.Parse()

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

		// Build the opencode command
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
		err := cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in iteration %d: %v\n", i, err)
			if i == iterations {
				os.Exit(1)
			}
			continue
		}

		// Check for completion from captured output
		output := outputBuffer.String()
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

EXAMPLES:
    ralph           # Run 10 iterations
    ralph 5         # Run 5 iterations  
    ralph -n 20 -v  # Run 20 iterations with verbose output
    ralph -debug    # Run with debug output (shows opencode logs)
    ralph -h        # Show this help

REQUIRED FILES:
    - tasks.md      Task definitions
    - progress.txt  Progress tracking
    - prompt.md     AI execution prompt

OPTIONAL FILES:
    - config.json   Custom prompt command

`)
}
