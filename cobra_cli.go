package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "giggum",
	Short: "AI-powered task management and agent execution system",
	Long: `Giggum is an AI-powered task management system that coordinates multiple agent types
to handle development tasks through intelligent agent execution.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior when no subcommand is specified
		// Parse iteration count from argument if provided
		if len(args) > 0 {
			var parsedIterations int
			_, err := fmt.Sscanf(args[0], "%d", &parsedIterations)
			if err == nil {
				iterations = parsedIterations
			}
		}

		// Run the main execution logic
		executeMain()
	},
}

// Global flags
var (
	verbose    bool
	debug      bool
	backup     bool
	restore    bool
	agentType  string
	timeout    int
	iterations int
)

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output (implies -v)")
	rootCmd.PersistentFlags().BoolVar(&backup, "backup", false, "Backup progress before running")
	rootCmd.PersistentFlags().BoolVar(&restore, "restore", false, "Restore progress from latest backup and exit")
	rootCmd.PersistentFlags().StringVar(&agentType, "agent", "backend-developer", "Specify agent type (tester, debugger, researcher, backend-developer, frontend-developer, ux, ui, marketer, feedbackseeker, simplifier, documentationwriter). Agent-only execution is enforced.")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 30, "Agent timeout in minutes (default: 30)")
	rootCmd.PersistentFlags().IntVarP(&iterations, "iterations", "n", 10, "Number of iterations to run")

	// Add subcommands
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(showProgressCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}

func executeMain() {
	// Debug implies verbose
	if debug {
		verbose = true
	}

	// Create logger
	logger, err := NewLogger("INFO", verbose, "")
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
	if agentType != "" {
		config.AgentType = AgentType(agentType)
	} else if config.AgentType == "" {
		// Ensure we have a default agent type if none was specified in config or CLI
		config.AgentType = BackendDeveloper
	}

	// Override timeout with command line value if provided
	if timeout > 0 {
		config.AgentTimeout = timeout
	}

	logger.Info("CLI: Agent-only execution enforced with %s agent (timeout: %d minutes)", config.AgentType, config.AgentTimeout)

	// Validate agent type - always required since we use agent-only execution
	if config.AgentType != "" {
		if _, err := GetAgentPrompt(config.AgentType); err != nil {
			logger.Error("Invalid agent type '%s': %v", config.AgentType, err)
			fmt.Fprintf(os.Stderr, "Error: Invalid agent type '%s'. This system requires agent-only execution.\n", config.AgentType)
			fmt.Fprintf(os.Stderr, "Use 'giggum agents list' to see available agent types.\n")
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
	if backup {
		if err := backupProgress(logger); err != nil {
			logger.Warn("Failed to backup progress: %v", err)
		}
	}

	// Run iterations
	runIterations(logger, config, iterations, debug)

	// Send webhook notification after completing all iterations
	if err := sendWebhookNotification(logger, config, iterations, true); err != nil {
		logger.Warn("Failed to send webhook notification: %v", err)
	}
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
