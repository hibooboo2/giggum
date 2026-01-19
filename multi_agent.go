package main

import (
	"fmt"
	"os"
	"strings"
)

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

	// Parse tasks from file
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
