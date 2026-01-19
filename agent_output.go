package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// runAgentWithOutput executes a task using a specific agent type and captures the output
func runAgentWithOutput(logger *Logger, agentType AgentType, task string, debug bool) (string, error) {
	// Get the current working directory for project path
	projectPath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %v", err)
	}

	// Initialize database manager
	dbPath := GetProjectDBPath(projectPath)
	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		logger.Warn("Failed to initialize database: %v", err)
		// Continue without database if it fails
	} else {
		defer dbManager.Close()
	}

	// Create agent session
	var sessionID int64
	if dbManager != nil {
		sessionID, err = dbManager.CreateAgentSession(agentType, projectPath)
		if err != nil {
			logger.Warn("Failed to create agent session: %v", err)
		}
	}

	// Get agent prompts (system and task)
	agentSystemPrompt, agentTaskPrompt, err := getAgentPrompts(agentType, task)
	if err != nil {
		return "", fmt.Errorf("failed to get agent prompts: %v", err)
	}

	logger.Info("Running %s agent for task: %s", agentType, task)

	// Build the opencode command
	args := []string{"run", "--model", "opencode/big-pickle"}
	if debug {
		args = append(args, "--print-logs")
	}

	// Combine the system prompt with the task prompt
	fullPrompt := fmt.Sprintf("%s\n\n%s", agentSystemPrompt, agentTaskPrompt)
	args = append(args, fullPrompt)

	cmd := exec.Command("opencode", args...)
	cmd.Env = append(os.Environ(), "OPENAI_BASE_URL=http://100.83.162.29:1234")

	// Set up output capture
	var outputBuffer bytes.Buffer
	var writer io.Writer

	if logger.verbose {
		// Use MultiWriter to capture output while displaying it
		writer = io.MultiWriter(&outputBuffer, os.Stdout)
	} else {
		writer = &outputBuffer
	}

	cmd.Stdout = writer
	cmd.Stderr = writer

	// Add input to database
	if dbManager != nil && sessionID > 0 {
		dbManager.AddSessionInput(sessionID, fullPrompt)
	}

	// Run the command
	err = cmd.Run()
	output := outputBuffer.String()

	// Add output to database
	if dbManager != nil && sessionID > 0 {
		dbManager.AddSessionOutput(sessionID, output)
	}

	if err != nil {
		return "", fmt.Errorf("agent execution failed: %v", err)
	}

	// Close the session
	if dbManager != nil && sessionID > 0 {
		dbManager.EndAgentSession(sessionID)
	}

	return output, nil
}

// parseResearcherOutput parses the structured output from the researcher agent
func parseResearcherOutput(output string) (title, description, status, priority, tags, metadata string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TITLE:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "TITLE:"))
		} else if strings.HasPrefix(line, "DESCRIPTION:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "DESCRIPTION:"))
		} else if strings.HasPrefix(line, "STATUS:") {
			status = strings.TrimSpace(strings.TrimPrefix(line, "STATUS:"))
		} else if strings.HasPrefix(line, "PRIORITY:") {
			priority = strings.TrimSpace(strings.TrimPrefix(line, "PRIORITY:"))
		} else if strings.HasPrefix(line, "TAGS:") {
			tags = strings.TrimSpace(strings.TrimPrefix(line, "TAGS:"))
		} else if strings.HasPrefix(line, "METADATA:") {
			metadata = strings.TrimSpace(strings.TrimPrefix(line, "METADATA:"))
		}
	}

	// Clean up title if it's too long
	if len(title) > 100 {
		title = title[:97] + "..."
	}

	return title, description, status, priority, tags, metadata
}
