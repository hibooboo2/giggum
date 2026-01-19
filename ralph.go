package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
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
	webServer    bool
	webPort      int
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
	flag.BoolVar(&args.webServer, "web-server", false, "Start web server for PWA")
	flag.IntVar(&args.webPort, "web-port", 8080, "Port for web server (default: 8080)")
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

func runIterations(logger *Logger, config Config, iterations int, debug bool) {
	// Agent-only execution - validate agent type exists
	if config.AgentType == "" {
		config.AgentType = BackendDeveloper
	}

	for i := 1; i <= iterations; i++ {
		fmt.Printf("Running iteration %d/%d with %s agent...\n", i, iterations, config.AgentType)

		// Execute using agent only - no traditional execution path
		err := runAgentWithOutputCapture(logger, config.AgentType, config.PromptCommand, debug)

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
			logger.Error("Agent type: %s, Task: %s", config.AgentType, config.PromptCommand)

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

	if args.webServer {
		startWebServer(args.webPort)
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
		if err := runMultiAgentSession(logger, args.iterations, args.debug); err != nil {
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
	runIterations(logger, *config, args.iterations, args.debug)

	// Send webhook notification after completing all iterations
	if err := sendWebhookNotification(logger, *config, args.iterations, true); err != nil {
		logger.Warn("Failed to send webhook notification: %v", err)
	}
}

// startWebServer starts the web server for PWA integration
func startWebServer(port int) {
	fmt.Printf("Starting giggum web server on port %d...\n", port)
	fmt.Printf("PWA will be available at: http://localhost:%d\n", port)
	fmt.Printf("API endpoints available at: http://localhost:%d/api\n", port)

	server := NewWebServer(port)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start web server: %v\n", err)
		os.Exit(1)
	}
}
