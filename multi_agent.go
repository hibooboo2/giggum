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

	// Enhanced keyword patterns with weighted scoring
	type agentScore struct {
		agent    AgentType
		score    int
		priority int
	}

	// Define comprehensive keyword patterns for each agent type
	agentPatterns := map[AgentType][]string{
		Tester: {
			"test", "testing", "tests", "unit test", "integration test", "e2e test", "quality",
			"defect", "defects", "validation", "verification", "qa", "quality assurance",
			"regression", "performance test", "load test", "stress test", "coverage",
			"tdd", "bdd", "assert", "mock", "test case", "test cases", "test suite",
		},
		Debugger: {
			"debug", "debugging", "fix", "fixes", "fixing", "error", "errors", "crash",
			"exception", "failure", "fail", "broken", "problem", "problems", "troubleshoot",
			"troubleshooting", "resolve", "resolution", "patch", "hotfix", "stack trace",
			"log", "logs", "investigate issue", "diagnose", "diagnosis", "root cause",
			"bug", "bugs", "buggy", "issue", "issues",
		},
		Researcher: {
			"research", "researching", "investigate", "investigation", "investigating", "study",
			"studies", "analyze", "analysis", "exploration", "explore", "survey", "benchmark",
			"compare", "comparison", "evaluate", "evaluation", "assess", "assessment",
			"review literature", "market research", "feasibility", "viability", "options",
		},
		BackendDeveloper: {
			"backend", "back end", "server", "server-side", "api", "apis", "rest", "graphql",
			"database", "databases", "sql", "nosql", "orm", "migration", "schema",
			"microservice", "microservices", "service", "services", "endpoint", "endpoints",
			"authentication", "authorization", "security", "middleware", "cache", "caching",
			"queue", "queues", "worker", "workers", "background job", "cron", "scheduled",
			"implement", "create", "build", "system", "architecture",
		},
		FrontendDeveloper: {
			"frontend", "front end", "client", "client-side", "component", "components",
			"view", "views", "page", "pages", "react", "vue", "angular", "javascript",
			"typescript", "css", "html", "web", "responsive", "mobile", "browser",
			"dom", "spa", "single page application",
		},
		UX: {
			"ux", "user experience", "user research", "usability", "user journey", "flow",
			"flows", "wireframe", "wireframes", "prototype", "prototypes", "persona",
			"personas", "user story", "user stories", "interaction", "interactions",
			"navigation", "usability test", "a/b test", "user testing", "behavior",
			"intuitive", "onboarding", "design", "designing",
		},
		UI: {
			"ui", "visual", "design", "layout", "styling", "theme", "themes", "color",
			"colors", "typography", "font", "fonts", "icon", "icons", "graphic",
			"graphics", "brand", "branding", "style guide", "design system", "aesthetic",
			"appearance", "look and feel", "visual hierarchy", "spacing", "grid",
		},
		DocumentationWriter: {
			"document", "documentation", "docs", "readme", "wiki", "guide", "guides",
			"manual", "manuals", "tutorial", "tutorials", "how to", "howto", "instructions",
			"reference", "api docs", "changelog", "release notes", "help", "help file",
			"knowledge base", "kb", "faq", "frequently asked", "writing", "technical writing",
		},
		Marketer: {
			"market", "marketing", "promote", "promotion", "content", "blog", "blog post",
			"article", "articles", "social media", "twitter", "facebook", "linkedin",
			"campaign", "campaigns", "advertise", "advertisement", "seo", "sem", "analytics",
			"metrics", "kpi", "growth", "engagement", "audience", "target", "strategy",
		},
		FeedbackSeeker: {
			"feedback", "review", "reviews", "interview", "interviews", "survey", "surveys",
			"questionnaire", "questionnaires", "poll", "polls", "user input", "collect feedback",
			"gather feedback", "user opinion", "customer", "customers", "stakeholder",
			"stakeholders", "suggestion", "suggestions", "improvement", "recommendations",
		},
		Simplifier: {
			"simplify", "simplifying", "explain", "explaining", "clarify", "clarification",
			"break down", "breakdown", "demystify", "make simple", "make understandable",
			"accessible", "easy to understand", "plain english", "layman's terms", "analogy",
			"analogies", "metaphor", "metaphors", "tutorial", "tutorialize", "educate",
		},
	}

	// Calculate scores for each agent type
	scores := make([]agentScore, 0, len(agentPatterns))

	// Create priority map for fallback ordering
	priorityMap := make(map[AgentType]int)
	for i, agent := range priority {
		priorityMap[agent] = i
	}

	// Score each agent based on keyword matches
	for agentType, keywords := range agentPatterns {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(taskLower, keyword) {
				// Weight longer keywords more heavily, but give extra weight to primary agent indicators
				weight := len(strings.Fields(keyword))

				// Bonus weight for specific primary keywords that strongly indicate agent type
				if agentType == Tester && (keyword == "test" || keyword == "testing") {
					weight += 2
				}
				if agentType == Researcher && (keyword == "research" || keyword == "analyze") {
					weight += 2
				}
				if agentType == UX && keyword == "ux" {
					weight += 2
				}

				score += weight
			}
		}
		if score > 0 {
			agentPriority := priorityMap[agentType]
			if agentPriority == 0 && len(priority) > 0 && agentType != priority[0] {
				// If not in priority list, give it the lowest priority
				agentPriority = len(priority)
			}
			scores = append(scores, agentScore{agentType, score, agentPriority})
		}
	}

	// If no matches found, use enhanced fallback logic
	if len(scores) == 0 {
		return selectBestAgentByContext(task, priority)
	}

	// Sort by score (descending), then by priority (ascending)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score < scores[j].score ||
				(scores[i].score == scores[j].score && scores[i].priority > scores[j].priority) {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	return scores[0].agent
}

// selectBestAgentByContext provides fallback logic for when no clear keyword match is found
func selectBestAgentByContext(task string, priority []AgentType) AgentType {
	taskLower := strings.ToLower(task)

	// Context-based heuristics for ambiguous tasks

	// Technical implementation tasks
	if containsAny(taskLower, []string{"implement", "create", "build", "develop", "code", "program"}) {
		if containsAny(taskLower, []string{"system", "architecture", "infrastructure", "scale"}) {
			return BackendDeveloper
		}
		if containsAny(taskLower, []string{"user", "customer", "interface", "experience"}) {
			return FrontendDeveloper
		}
		return BackendDeveloper // Default to backend for general implementation
	}

	// Planning and strategy tasks
	if containsAny(taskLower, []string{"plan", "strategy", " roadmap", "design", "architecture"}) {
		if containsAny(taskLower, []string{"user", "customer", "experience", "journey", "ux"}) {
			return UX
		}
		if containsAny(taskLower, []string{"visual", "look", "feel", "brand"}) {
			return UI
		}
		if containsAny(taskLower, []string{"system", "technical", "infrastructure"}) {
			return BackendDeveloper
		}
		return Researcher
	}

	// Content and communication tasks
	if containsAny(taskLower, []string{"write", "content", "text", "copy", "message"}) {
		if containsAny(taskLower, []string{"technical", "api", "documentation", "instructions"}) {
			return DocumentationWriter
		}
		if containsAny(taskLower, []string{"marketing", "sales", "promotion", "campaign"}) {
			return Marketer
		}
		return DocumentationWriter
	}

	// Analysis and improvement tasks
	if containsAny(taskLower, []string{"improve", "optimize", "enhance", "refactor", "update"}) {
		if containsAny(taskLower, []string{"performance", "speed", "efficiency", "scale"}) {
			return BackendDeveloper
		}
		if containsAny(taskLower, []string{"experience", "usability", "accessibility"}) {
			return UX
		}
		return BackendDeveloper
	}

	// Default fallback with smart prioritization
	if len(priority) > 0 {
		// For completely ambiguous tasks, prefer generalist agents
		for _, preferredAgent := range []AgentType{BackendDeveloper, Researcher, FrontendDeveloper} {
			for _, agent := range priority {
				if agent == preferredAgent {
					return agent
				}
			}
		}
		return priority[0]
	}

	// Ultimate fallback
	return BackendDeveloper
}

// containsAny checks if the text contains any of the provided substrings
func containsAny(text string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(text, substr) {
			return true
		}
	}
	return false
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
    -agent TYPE     Use specific agent type (default: backend-developer, see -list-agents for available types)
    -multi-agent    Run coordinated multi-agent session
    -list-agents    List all available agent types and their descriptions
    -show-progress  Show agent progress for current project

 EXAMPLES:
    ralph           # Run 10 iterations with default backend-developer agent
    ralph 5         # Run 5 iterations with default backend-developer agent
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
