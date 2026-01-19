package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
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
	// Future configuration options can be added here
}

// DefaultPromptCommand is the default prompt command if not specified in config
const DefaultPromptCommand = "@tasks.md @progress.txt @prompt.md execute the prompt in prompt.md @tasks.md @progress.txt @prompt.md execute the prompt in prompt.md"

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
		return &Config{PromptCommand: DefaultPromptCommand}, nil
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

	logger.Debug("Loaded configuration from '%s'", configFile)

	return &config, nil
}

func main() {
	// Define command line flags
	var help bool
	var verbose bool
	var debug bool
	var iterations int
	var backup bool
	var restore bool
	var logLevel string
	var logFile string

	flag.BoolVar(&help, "h", false, "Show help message")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&debug, "debug", false, "Enable debug output (implies -v)")
	flag.IntVar(&iterations, "n", 10, "Number of iterations to run")
	flag.BoolVar(&backup, "backup", false, "Create backup of progress.txt before starting")
	flag.BoolVar(&restore, "restore", false, "Restore progress.txt from latest backup")
	flag.StringVar(&logLevel, "log-level", "INFO", "Set logging level (DEBUG, INFO, WARN, ERROR)")
	flag.StringVar(&logFile, "log-file", "", "Write logs to file (default: stdout only)")
	flag.Parse()

	// Show help if requested
	if help {
		showHelp()
		return
	}

	// Set up logging
	debug = debug || verbose
	if debug {
		verbose = true
		logLevel = "DEBUG"
	}

	// Create logger
	logger, err := NewLogger(logLevel, verbose, logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Ralph Wiggum starting up", "iterations", iterations, "logLevel", logLevel, "logFile", logFile)

	// Handle restore flag
	if restore {
		logger.Info("Restoring progress from backup")
		if err := restoreProgress(logger); err != nil {
			logger.Error("Restore failed: %v", err)
			os.Exit(1)
		}
		logger.Info("Restore completed successfully")
		return
	}

	// Create backup if requested
	if backup {
		logger.Info("Creating backup of progress.txt")
		if err := backupProgress(logger); err != nil {
			logger.Error("Backup failed: %v", err)
			os.Exit(1)
		}
		logger.Info("Backup created successfully")
	}

	// Load configuration
	logger.Debug("Loading configuration")
	config, err := loadConfig(logger)
	if err != nil {
		logger.Error("Configuration loading failed: %v", err)
		os.Exit(1)
	}
	logger.Debug("Configuration loaded successfully")

	// Enhanced feedback loop validation
	logger.Debug("Validating feedback loops")
	if err := validateFeedbackLoops(logger); err != nil {
		logger.Error("Feedback loop validation failed: %v", err)
		os.Exit(1)
	}
	logger.Debug("Feedback loop validation passed")

	// Parse remaining arguments for iteration count
	if flag.NArg() > 0 {
		var parsedIterations int
		_, err := fmt.Sscanf(flag.Arg(0), "%d", &parsedIterations)
		if err == nil {
			iterations = parsedIterations
		}
	}

	// Run the autonomous coding loop
	for i := 1; i <= iterations; i++ {
		fmt.Printf("Running iteration %d/%d...\n", i, iterations)

		// Build the opencode command
		args := []string{"run"}
		if verbose {
			args = append(args, "--print-logs")
		}
		args = append(args, config.PromptCommand)

		cmd := exec.Command("opencode", args...)

		// Inherit environment variables from current process
		cmd.Env = os.Environ()

		// Capture output
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running opencode on iteration %d: %v\n", i, err)
			if verbose {
				fmt.Fprintf(os.Stderr, "Command output: %s\n", string(output))
			}

			// Check if it's a network connectivity issue
			if strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "network") {
				fmt.Fprintf(os.Stderr, "Network error detected. Please check your internet connection.\n")
			}

			// Continue to next iteration instead of exiting immediately
			if i == iterations {
				fmt.Fprintf(os.Stderr, "Final iteration failed. Exiting.\n")
				os.Exit(1)
			}
			continue
		}

		result := string(output)

		// Check if we got any output
		if len(result) == 0 {
			fmt.Printf("Warning: No output received from opencode on iteration %d\n", i)
			continue
		}

		if verbose {
			fmt.Printf("Opencode output received (%d bytes)\n", len(result))
		}

		fmt.Println(result)

		// Check for completion signal
		if strings.Contains(result, "<promise>COMPLETE</promise>") || strings.Contains(result, "✅ Complete ✅") {
			fmt.Println("✅ All tasks completed!")
			os.Exit(0)
		}

		// Check for explicit error messages in output
		if strings.Contains(strings.ToLower(result), "error") && !strings.Contains(result, "error handling") {
			if verbose {
				fmt.Printf("Warning: Potential error detected in opencode output on iteration %d\n", i)
			}
		}
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
	fmt.Printf(`Ralph Wiggum - Autonomous AI Coding Loop Executor

USAGE:
    ralph [OPTIONS] [ITERATIONS]

ARGUMENTS:
    ITERATIONS    Number of iterations to run (default: 10)

OPTIONS:
    -h, --help      Show this help message
    -v              Enable verbose output
    -debug          Enable debug output (implies -v)
    -n N            Set number of iterations (default: 10)
    -backup         Create backup of progress.txt before starting
    -restore        Restore progress.txt from latest backup and exit
    -log-level LVL  Set logging level (DEBUG, INFO, WARN, ERROR)
    -log-file FILE  Write logs to file (default: stdout only)

DESCRIPTION:
    This program runs an autonomous AI coding loop using the opencode CLI.
    It executes the prompt defined in prompt.md, tracks progress in progress.txt,
    and manages tasks defined in tasks.md.

    The loop continues until all tasks are complete or the specified number 
    of iterations is reached.

CONFIGURATION:
    The program looks for a config.json file for customization. If not found,
    default settings are used. The config file can contain:
    
    {
        "prompt_command": "your custom prompt here"
    }

BACKUP/RESTORE:
    The -backup flag creates a timestamped backup of progress.txt in the 
    'backups/' directory before starting the loop. The -restore flag restores
    progress.txt from the most recent backup and exits.

FILES REQUIRED:
    - tasks.md     Task definitions and priorities
    - progress.txt Progress tracking
    - prompt.md    AI execution prompt
    - config.json  Optional configuration file

EXAMPLES:
    ralph                    # Run 10 iterations
    ralph 5                  # Run 5 iterations
    ralph -n 20 -v           # Run 20 iterations with verbose output
    ralph -debug             # Run with debug logging
    ralph -log-level DEBUG   # Run with DEBUG level logging
    ralph -log-file ralph.log # Run and log to file
    ralph -backup            # Create backup and run 10 iterations
    ralph -restore           # Restore from latest backup
    ralph -h                 # Show help

`)
}
