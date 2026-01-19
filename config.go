package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Config holds the configuration for the Ralph Wiggum loop
// Note: This system uses AGENT-ONLY execution - no traditional execution paths exist
type Config struct {
	PromptCommand string    `json:"prompt_command"`
	WebhookURL    string    `json:"webhook_url"`
	ParseReply    bool      `json:"wait_for_reply"`
	ReplyPrompt   string    `json:"reply_prompt"`
	AddToTasks    bool      `json:"add_to_tasks"`
	AgentType     AgentType `json:"agent_type"`
	UseAgents     bool      `json:"use_agents"` // DEPRECATED: Always true for agent-only execution
	// Future configuration options can be added here
}

// DefaultPromptCommand is the default prompt command if not specified in config
const DefaultPromptCommand = "@tasks.md @progress.txt @prompt.md Follow the instrunctions in prompt.md"

// loadConfig loads configuration from config.json file, returns default config if file doesn't exist
func loadConfig(logger *Logger) (*Config, error) {
	configFile := "config.json"

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		logger.Debug("Config file '%s' not found, using default configuration", configFile)
		return &Config{
			PromptCommand: DefaultPromptCommand,
			ParseReply:    false,
			ReplyPrompt:   "Enter your response (or press Enter to continue): ",
			AddToTasks:    true,
			UseAgents:     true,             // Agent-only execution enforced
			AgentType:     BackendDeveloper, // Default to BackendDeveloper
		}, nil
	}

	// Read config file
	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %v", configFile, err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %v", configFile, err)
	}

	// Use default prompt command if not specified
	if config.PromptCommand == "" {
		config.PromptCommand = DefaultPromptCommand
	}

	// Set defaults for reply functionality
	if config.ReplyPrompt == "" {
		config.ReplyPrompt = "Enter your response (or press Enter to continue): "
	}

	// Enforce agent-only execution - no traditional execution paths
	if config.AgentType == "" {
		config.AgentType = BackendDeveloper // Default to backend developer
	}
	// Force agent-only execution - traditional execution is not supported
	config.UseAgents = true

	logger.Info("Agent-only execution enforced: %s agent selected", config.AgentType)

	logger.Debug("Loaded configuration from '%s'", configFile)

	return &config, nil
}

// validateFeedbackLoops validates required files and tools
func validateFeedbackLoops(logger *Logger) error {
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
			logger.Warn("File '%s' is empty. Consider adding content to ensure proper operation.", file)
		}

		logger.Debug("File '%s' exists and is accessible (%d bytes)", file, info.Size())
	}

	logger.Debug("Feedback loop validation passed")
	logger.Debug("opencode CLI is available")
	logger.Debug("All required files exist and are accessible")

	return nil
}
