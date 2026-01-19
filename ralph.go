package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Define command line flags
	var help bool
	var verbose bool
	var iterations int

	flag.BoolVar(&help, "h", false, "Show help message")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.BoolVar(&verbose, "v", false, "Enable verbose/debug output")
	flag.IntVar(&iterations, "n", 10, "Number of iterations to run")
	flag.Parse()

	// Show help if requested
	if help {
		showHelp()
		return
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
			fmt.Fprintf(os.Stderr, "Error running opencode: %v\n", err)
			os.Exit(1)
		}

		result := string(output)
		fmt.Println(result)

		// Check for completion signal
		if strings.Contains(result, "<promise>COMPLETE</promise>") {
			fmt.Println("PRD complete, exiting.")
			os.Exit(0)
		}
	}
}

func validateFeedbackLoops(verbose bool) error {
	// Check if opencode CLI is available
	if _, err := exec.LookPath("opencode"); err != nil {
		return fmt.Errorf("opencode CLI not found: %v", err)
	}

	// Validate required files exist
	requiredFiles := []string{"tasks.md", "progress.txt", "prompt.md"}
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("required file %s not found", file)
		}
	}

	if verbose {
		fmt.Println("✓ Feedback loop validation passed")
		fmt.Println("✓ opencode CLI is available")
		fmt.Println("✓ All required files exist")
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

DESCRIPTION:
    This program runs an autonomous AI coding loop using the opencode CLI.
    It executes the prompt defined in prompt.md, tracks progress in progress.txt,
    and manages tasks defined in tasks.md.

    The loop continues until all tasks are complete or the specified number 
    of iterations is reached.

FILES REQUIRED:
    - tasks.md     Task definitions and priorities
    - progress.txt Progress tracking
    - prompt.md    AI execution prompt

EXAMPLES:
    ralph           # Run 10 iterations
    ralph 5         # Run 5 iterations
    ralph -n 20 -v  # Run 20 iterations with verbose output
    ralph -h        # Show help

`)
}
