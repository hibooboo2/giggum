package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Define command line flags
	var help bool
	var verbose bool
	var iterations int
	var backup bool
	var restore bool

	flag.BoolVar(&help, "h", false, "Show help message")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.BoolVar(&verbose, "v", false, "Enable verbose/debug output")
	flag.IntVar(&iterations, "n", 10, "Number of iterations to run")
	flag.BoolVar(&backup, "backup", false, "Create backup of progress.txt before starting")
	flag.BoolVar(&restore, "restore", false, "Restore progress.txt from latest backup")
	flag.Parse()

	// Show help if requested
	if help {
		showHelp()
		return
	}

	// Handle restore flag
	if restore {
		if err := restoreProgress(verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Restore failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create backup if requested
	if backup {
		if err := backupProgress(verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Enhanced feedback loop validation
	if err := validateFeedbackLoops(verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Feedback loop validation failed: %v\n", err)
		os.Exit(1)
	}

	// Parse remaining arguments for iteration count
	if flag.NArg() > 0 {
		var parsedIterations int
		_, err := fmt.Sscanf(flag.Arg(0), "%d", &parsedIterations)
		if err == nil {
			iterations = parsedIterations
		}
	}

	// Run the autonomous coding loop
	for i := 1; i < iterations; i++ {
		if verbose {
			fmt.Printf("\n--- Iteration %d/%d ---\n", i, iterations-1)
		}

		// Build the opencode command
		args := []string{"run"}
		if verbose {
			args = append(args, "--print-logs")
		}
		args = append(args, "--model", "opencode/big-pickle", "@tasks.md @progress.txt @prompt.md execute the prompt in prompt.md @tasks.md @progress.txt @prompt.md execute the prompt in prompt.md")

		cmd := exec.Command("opencode", args...)

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
			if i == iterations-1 {
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
			fmt.Println("Task complete, exiting.")
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

func validateFeedbackLoops(verbose bool) error {
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
			fmt.Printf("Warning: File '%s' is empty. Consider adding content to ensure proper operation.\n", file)
		}

		if verbose {
			fmt.Printf("✓ File '%s' exists and is accessible (%d bytes)\n", file, info.Size())
		}
	}

	if verbose {
		fmt.Println("✓ Feedback loop validation passed")
		fmt.Println("✓ opencode CLI is available")
		fmt.Println("✓ All required files exist and are accessible")
	}

	return nil
}

// backupProgress creates a timestamped backup of progress.txt
func backupProgress(verbose bool) error {
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

	if verbose {
		fmt.Printf("✓ Created backup: %s\n", backupFile)
	}

	return nil
}

// restoreProgress restores progress.txt from the latest backup
func restoreProgress(verbose bool) error {
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

	if verbose {
		fmt.Printf("✓ Restored from backup: %s\n", latestFile)
	} else {
		fmt.Printf("Restored progress.txt from: %s\n", latestFile)
	}

	return nil
}

func showHelp() {
	fmt.Printf(`Ralph Wiggum - Autonomous AI Coding Loop Executor

USAGE:
    ralph [OPTIONS] [ITERATIONS]

ARGUMENTS:
    ITERATIONS    Number of iterations to run (default: 10)

OPTIONS:
    -h, --help    Show this help message
    -v            Enable verbose/debug output
    -n N          Set number of iterations (default: 10)
    -backup       Create backup of progress.txt before starting
    -restore      Restore progress.txt from latest backup and exit

DESCRIPTION:
    This program runs an autonomous AI coding loop using the opencode CLI.
    It executes the prompt defined in prompt.md, tracks progress in progress.txt,
    and manages tasks defined in tasks.md.

    The loop continues until all tasks are complete or the specified number 
    of iterations is reached.

BACKUP/RESTORE:
    The -backup flag creates a timestamped backup of progress.txt in the 
    'backups/' directory before starting the loop. The -restore flag restores
    progress.txt from the most recent backup and exits.

FILES REQUIRED:
    - tasks.md     Task definitions and priorities
    - progress.txt Progress tracking
    - prompt.md    AI execution prompt

EXAMPLES:
    ralph           # Run 10 iterations
    ralph 5         # Run 5 iterations
    ralph -n 20 -v  # Run 20 iterations with verbose output
    ralph -backup   # Create backup and run 10 iterations
    ralph -restore  # Restore from latest backup
    ralph -h        # Show help

`)
}
