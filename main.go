package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	args := parseFlags()

	dbPath := GetTaskDBPath()
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

	// Override timeout with command line value if provided
	if args.timeout > 0 {
		config.AgentTimeout = args.timeout
	}

	logger.Info("CLI: Agent-only execution enforced with %s agent (timeout: %d minutes)", config.AgentType, config.AgentTimeout)

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
