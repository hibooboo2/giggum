package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// createAgentPrompt creates a prompt for a specific agent type
func createAgentPrompt(agentType AgentType, task string) (string, error) {
	_, taskPrompt, err := getAgentPrompts(agentType, task)
	if err != nil {
		return "", fmt.Errorf("failed to get agent prompt: %v", err)
	}
	return taskPrompt, nil
}

// getAgentPrompts gets both system and task prompts for a specific agent type
func getAgentPrompts(agentType AgentType, task string) (string, string, error) {
	// Try to get the current working directory for database access
	projectPath, err := os.Getwd()
	if err != nil {
		// Fallback to hardcoded prompts if we can't get current directory
		prompt, err := GetAgentPrompt(agentType)
		if err != nil {
			return "", "", fmt.Errorf("failed to get agent prompt: %v", err)
		}
		return prompt.SystemPrompt, fmt.Sprintf(prompt.TaskPrompt, task), nil
	}

	// Try to get prompt from database
	dbPath := GetProjectDBPath(projectPath)
	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		// Fallback to hardcoded prompts if database fails
		prompt, err := GetAgentPrompt(agentType)
		if err != nil {
			return "", "", fmt.Errorf("failed to get agent prompt: %v", err)
		}
		return prompt.SystemPrompt, fmt.Sprintf(prompt.TaskPrompt, task), nil
	}
	defer dbManager.Close()

	// Try to get stored prompt from database
	prompt, err := dbManager.GetStoredAgentPrompt(agentType)
	if err != nil {
		// Fallback to hardcoded prompts if not found in database
		prompt, err := GetAgentPrompt(agentType)
		if err != nil {
			return "", "", fmt.Errorf("failed to get agent prompt: %v", err)
		}
		return prompt.SystemPrompt, fmt.Sprintf(prompt.TaskPrompt, task), nil
	}

	return prompt.SystemPrompt, fmt.Sprintf(prompt.TaskPrompt, task), nil
}

// runAgent executes a task using a specific agent type
func runAgent(logger *Logger, agentType AgentType, task string, debug bool, timeoutMinutes int) error {
	// Get the current working directory for project path
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
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
		return fmt.Errorf("failed to get agent prompts: %v", err)
	}

	logger.Info("Running %s agent for task: %s", agentType, task)

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	// Build the opencode command
	args := []string{"run", "--model", "opencode/big-pickle"}
	if debug {
		args = append(args, "--print-logs")
	}

	// Add timeout information to the task prompt
	timeoutNotice := fmt.Sprintf("\n\nIMPORTANT: You have %d minutes maximum to complete this task. Monitor your time and ensure you reach a stopping point before the timeout. If you're running short on time, prioritize the most critical aspects and provide a partial solution with clear next steps.", timeoutMinutes)
	enhancedTaskPrompt := agentTaskPrompt + timeoutNotice

	// Combine the system prompt with the enhanced task prompt
	fullPrompt := fmt.Sprintf("%s\n\n%s", agentSystemPrompt, enhancedTaskPrompt)
	args = append(args, fullPrompt)

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "OPENAI_BASE_URL=http://100.83.162.29:1234")

	// Set up output capture
	var outputBuffer bytes.Buffer
	var writer io.Writer = &outputBuffer

	writer = io.MultiWriter(&outputBuffer, os.Stdout)

	cmd.Stdout = writer
	cmd.Stderr = writer

	// Add input to database
	if dbManager != nil && sessionID > 0 {
		dbManager.AddSessionInput(sessionID, fullPrompt)
	}

	// Run the command with timeout
	err = cmd.Run()
	output := outputBuffer.String()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("agent execution timed out after %d minutes", timeoutMinutes)
	}

	// Add output to database
	if dbManager != nil && sessionID > 0 {
		dbManager.AddSessionOutput(sessionID, output)
	}

	if err != nil {
		return fmt.Errorf("agent execution failed: %v", err)
	}

	// Close the session
	if dbManager != nil && sessionID > 0 {
		dbManager.EndAgentSession(sessionID)
	}

	return nil
}

// runAgentWithOutputCapture executes a task using a specific agent type and captures output for completion detection
func runAgentWithOutputCapture(logger *Logger, agentType AgentType, task string, debug bool, timeoutMinutes int) error {
	// Get the current working directory for project path
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
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
		return fmt.Errorf("failed to get agent prompts: %v", err)
	}

	logger.Info("Running %s agent for task: %s", agentType, task)

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	// Build the opencode command
	args := []string{"run", "--model", "opencode/big-pickle"}
	if debug {
		args = append(args, "--print-logs")
	}

	// Add timeout information to the task prompt
	timeoutNotice := fmt.Sprintf("\n\nIMPORTANT: You have %d minutes maximum to complete this task. Monitor your time and ensure you reach a stopping point before the timeout. If you're running short on time, prioritize the most critical aspects and provide a partial solution with clear next steps.", timeoutMinutes)
	enhancedTaskPrompt := agentTaskPrompt + timeoutNotice

	// Combine the system prompt with the enhanced task prompt
	fullPrompt := fmt.Sprintf("%s\n\n%s", agentSystemPrompt, enhancedTaskPrompt)
	args = append(args, fullPrompt)

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "OPENAI_BASE_URL=http://100.83.162.29:1234")

	// Set up output capture for completion detection and verbose output
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

	// Run the command with timeout
	err = cmd.Run()
	output := outputBuffer.String()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("agent execution timed out after %d minutes", timeoutMinutes)
	}

	// Add output to database
	if dbManager != nil && sessionID > 0 {
		dbManager.AddSessionOutput(sessionID, output)
	}

	if err != nil {
		return fmt.Errorf("agent execution failed: %v", err)
	}

	// Check for completion from captured output
	if strings.Contains(output, "✅ Complete ✅") {
		fmt.Println("✅ All tasks completed!")
		// Return a special error to indicate completion for proper handling in calling function
		return fmt.Errorf("COMPLETED")
	}

	// Close the session
	if dbManager != nil && sessionID > 0 {
		dbManager.EndAgentSession(sessionID)
	}

	return nil
}
