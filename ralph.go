package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
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
	useAgents    bool
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
	flag.StringVar(&args.agentType, "agent", "", "Specify agent type (tester, debugger, researcher, backend-developer, frontend-developer, ux, ui, marketer, feedbackseeker, simplifier, documentationwriter)")
	flag.BoolVar(&args.useAgents, "use-agents", false, "Enable multi-agent mode")
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

			// Set up output capture for tee reader functionality
			var outputBuffer bytes.Buffer
			var writer io.Writer = &outputBuffer

			if logger.verbose {
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
			if err := sendWebhookNotification(logger, config, i, true); err != nil {
				logger.Warn("Failed to send webhook notification: %v", err)
			}
			return // Exit early since tasks are complete
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
	if args.useAgents {
		config.UseAgents = true
	}
	if args.agentType != "" {
		config.AgentType = AgentType(args.agentType)
		config.UseAgents = true
	}

	// Handle multi-agent mode
	if args.multiAgent {
		if err := runMultiAgentSession(logger, args.iterations, args.debug); err != nil {
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
