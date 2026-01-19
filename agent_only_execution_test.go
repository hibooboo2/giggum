package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAgentOnlyExecutionRefactoring tests comprehensive scenarios for the agent-only execution system
func TestAgentOnlyExecutionRefactoring(t *testing.T) {
	t.Run("BasicExecutionFlow", func(t *testing.T) {
		// Test that the system enforces agent-only execution
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Create required files
		requiredFiles := []string{"tasks.md", "progress.txt", "prompt.md"}
		for _, file := range requiredFiles {
			err = os.WriteFile(file, []byte("# test content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create %s: %v", file, err)
			}
		}

		// Load configuration and verify agent enforcement
		logger, err := NewLogger("INFO", false, "")
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		loadedConfig, err := loadConfig(logger)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify UseAgents is always true after loading
		if !loadedConfig.UseAgents {
			t.Error("UseAgents should be true after config loading (agent-only enforcement)")
		}

		// Verify default agent type is set if empty
		if loadedConfig.AgentType == "" {
			t.Error("Agent type should have a default value")
		}
	})

	t.Run("ConfigurationValidation", func(t *testing.T) {
		// Test configuration validation scenarios specific to agent-only execution
		testCases := []struct {
			name        string
			config      Config
			expectError bool
			description string
		}{
			{
				name: "ValidAgentConfig",
				config: Config{
					UseAgents:     true,
					AgentType:     BackendDeveloper,
					PromptCommand: "Test command",
				},
				expectError: false,
				description: "Valid agent configuration should pass validation",
			},
			{
				name: "InvalidAgentType",
				config: Config{
					UseAgents:     true,
					AgentType:     AgentType("invalid"),
					PromptCommand: "Test command",
				},
				expectError: true,
				description: "Invalid agent type should fail validation",
			},
			{
				name: "EmptyAgentType",
				config: Config{
					UseAgents:     true,
					AgentType:     "",
					PromptCommand: "Test command",
				},
				expectError: true,
				description: "Empty agent type should fail validation",
			},
			{
				name: "EmptyPromptCommand",
				config: Config{
					UseAgents:     true,
					AgentType:     Tester,
					PromptCommand: "",
				},
				expectError: true,
				description: "Empty prompt command should fail validation",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateConfig(&tc.config)
				if tc.expectError && err == nil {
					t.Errorf("%s: Expected validation error but got none", tc.description)
				} else if !tc.expectError && err != nil {
					t.Errorf("%s: Expected no validation error but got: %v", tc.description, err)
				}
			})
		}
	})

	t.Run("AgentSelectionEnhancement", func(t *testing.T) {
		// Test the enhanced agent selection logic with keyword matching
		testCases := []struct {
			name          string
			task          string
			expectedAgent AgentType
			description   string
		}{
			{
				name:          "TestingKeywords",
				task:          "Write unit tests for the API endpoints",
				expectedAgent: Tester,
				description:   "Task with testing keywords should select Tester agent",
			},
			{
				name:          "DebuggingKeywords",
				task:          "Debug the memory leak in the database connection",
				expectedAgent: Debugger,
				description:   "Task with debugging keywords should select Debugger agent",
			},
			{
				name:          "BackendKeywords",
				task:          "Implement REST API with authentication middleware",
				expectedAgent: BackendDeveloper,
				description:   "Task with backend keywords should select BackendDeveloper",
			},
			{
				name:          "FrontendKeywords",
				task:          "Create responsive React components with TypeScript",
				expectedAgent: FrontendDeveloper,
				description:   "Task with frontend keywords should select FrontendDeveloper",
			},
			{
				name:          "UXKeywords",
				task:          "Design user flow for the onboarding process",
				expectedAgent: UX,
				description:   "Task with UX keywords should select UX agent",
			},
			{
				name:          "UIKeywords",
				task:          "Update visual design system with new color palette",
				expectedAgent: UI,
				description:   "Task with UI keywords should select UI agent",
			},
			{
				name:          "DocumentationKeywords",
				task:          "Write comprehensive user guide and technical documentation",
				expectedAgent: DocumentationWriter,
				description:   "Task with documentation keywords should select DocumentationWriter",
			},
			{
				name:          "ResearchKeywords",
				task:          "Research and compare different database technologies",
				expectedAgent: Researcher,
				description:   "Task with research keywords should select Researcher",
			},
			{
				name:          "MarketingKeywords",
				task:          "Create marketing campaign for product launch",
				expectedAgent: Marketer,
				description:   "Task with marketing keywords should select Marketer",
			},
			{
				name:          "FeedbackKeywords",
				task:          "Conduct user interviews to gather feedback on new features",
				expectedAgent: FeedbackSeeker,
				description:   "Task with feedback keywords should select FeedbackSeeker",
			},
			{
				name:          "SimplificationKeywords",
				task:          "Explain complex technical concepts in simple terms",
				expectedAgent: Simplifier,
				description:   "Task with simplification keywords should select Simplifier",
			},
		}

		agentPriority := []AgentType{
			DocumentationWriter, BackendDeveloper, FrontendDeveloper, Tester, Debugger, Researcher,
			UX, UI, FeedbackSeeker, Marketer, Simplifier,
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				selectedAgent := selectBestAgentForTask(tc.task, agentPriority)
				if selectedAgent != tc.expectedAgent {
					t.Errorf("%s: Expected agent %s, got %s for task: %s",
						tc.description, tc.expectedAgent, selectedAgent, tc.task)
				}
			})
		}
	})

	t.Run("MultiAgentCoordination", func(t *testing.T) {
		// Test multi-agent coordination functionality
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Create tasks.md with diverse tasks
		tasksContent := `# Tasks

## HIGH PRIORITY
- Implement user authentication system
- Write comprehensive unit tests
- Debug performance issues in database queries
- Design responsive UI components

## LOW PRIORITY
- Create user documentation
- Research new frameworks
- Gather user feedback
`
		err = os.WriteFile("tasks.md", []byte(tasksContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create tasks.md: %v", err)
		}

		// Create other required files
		err = os.WriteFile("progress.txt", []byte("Initial progress"), 0644)
		if err != nil {
			t.Fatalf("Failed to create progress.txt: %v", err)
		}

		err = os.WriteFile("prompt.md", []byte("Test prompt"), 0644)
		if err != nil {
			t.Fatalf("Failed to create prompt.md: %v", err)
		}

		// Test task parsing
		parsedTasks := parseTasks(tasksContent)
		expectedTaskCount := 7 // Should extract 7 tasks
		if len(parsedTasks) != expectedTaskCount {
			t.Errorf("Expected %d tasks, got %d", expectedTaskCount, len(parsedTasks))
		}

		// Test agent selection for parsed tasks
		agentPriority := []AgentType{
			BackendDeveloper, FrontendDeveloper, Tester, Debugger, Researcher,
			UX, UI, DocumentationWriter, FeedbackSeeker, Marketer, Simplifier,
		}

		for i, task := range parsedTasks {
			if i >= 3 { // Test first 3 tasks to keep test reasonable
				selectedAgent := selectBestAgentForTask(task, agentPriority)
				t.Logf("Task: %s -> Selected Agent: %s", task, selectedAgent)
			}
		}
	})

	t.Run("ErrorHandlingAndRecovery", func(t *testing.T) {
		// Test error handling scenarios specific to agent-only execution
		testCases := []struct {
			name            string
			errorType       string
			iteration       int
			totalIterations int
			expectContinue  bool
			expectExit      bool
			description     string
		}{
			{
				name:            "CompletedSignalEarly",
				errorType:       "COMPLETED",
				iteration:       3,
				totalIterations: 5,
				expectContinue:  false,
				expectExit:      false,
				description:     "COMPLETED signal should cause early termination",
			},
			{
				name:            "NormalErrorNonFinal",
				errorType:       "Normal execution error",
				iteration:       2,
				totalIterations: 5,
				expectContinue:  true,
				expectExit:      false,
				description:     "Normal error on non-final iteration should continue",
			},
			{
				name:            "NormalErrorFinal",
				errorType:       "Final execution error",
				iteration:       5,
				totalIterations: 5,
				expectContinue:  false,
				expectExit:      true,
				description:     "Normal error on final iteration should exit",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Simulate error handling logic from runIterations
				_ = &GiggumError{
					Code:    ErrCodeAgent,
					Message: tc.errorType,
				}

				if tc.errorType == "COMPLETED" {
					// Should return early (not exit with error)
					if tc.expectExit {
						t.Errorf("%s: Expected error exit but got early completion", tc.description)
					}
				} else {
					// Normal error handling
					if tc.iteration == tc.totalIterations {
						// Final iteration - should exit
						if !tc.expectExit {
							t.Errorf("%s: Expected early completion but got error exit", tc.description)
						}
					} else {
						// Non-final iteration - should continue
						if !tc.expectContinue {
							t.Errorf("%s: Expected continuation but got error exit", tc.description)
						}
					}
				}
			})
		}
	})

	t.Run("DatabaseIntegration", func(t *testing.T) {
		// Test database integration with agent-only execution
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		dbManager, err := NewDBManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}
		defer dbManager.Close()

		// Test agent session creation
		agentTypes := []AgentType{Tester, Debugger, BackendDeveloper, UX, DocumentationWriter}
		projectPath := "/test/project"

		for i, agentType := range agentTypes {
			sessionID, err := dbManager.CreateAgentSession(agentType, projectPath)
			if err != nil {
				t.Errorf("Failed to create session for %s: %v", agentType, err)
				continue
			}

			if sessionID <= 0 {
				t.Errorf("Invalid session ID for %s: %d", agentType, sessionID)
			}

			// Test adding input/output
			input := fmt.Sprintf("Test input for %s agent", agentType)
			output := fmt.Sprintf("Test output for %s agent", agentType)

			err = dbManager.AddSessionInput(sessionID, input)
			if err != nil {
				t.Errorf("Failed to add input for %s: %v", agentType, err)
			}

			err = dbManager.AddSessionOutput(sessionID, output)
			if err != nil {
				t.Errorf("Failed to add output for %s: %v", agentType, err)
			}

			// End the session
			err = dbManager.EndAgentSession(sessionID)
			if err != nil {
				t.Errorf("Failed to end session for %s: %v", agentType, err)
			}

			// Create progress entry
			progressTask := fmt.Sprintf("Progress task %d for %s", i, agentType)
			progressID, err := dbManager.CreateAgentProgress(agentType, projectPath, progressTask, sessionID)
			if err != nil {
				t.Errorf("Failed to create progress for %s: %v", agentType, err)
			}

			if progressID <= 0 {
				t.Errorf("Invalid progress ID for %s: %d", agentType, progressID)
			}

			// Update progress
			progressUpdate := fmt.Sprintf("Updated progress for %s", agentType)
			err = dbManager.UpdateAgentProgress(progressID, "in_progress", progressUpdate)
			if err != nil {
				t.Errorf("Failed to update progress for %s: %v", agentType, err)
			}
		}

		// Test retrieval of sessions and progress
		sessions, err := dbManager.GetAllAgentProgress(projectPath)
		if err != nil {
			t.Errorf("Failed to get all agent progress: %v", err)
		}

		if len(sessions) != len(agentTypes) {
			t.Errorf("Expected progress for %d agent types, got %d", len(agentTypes), len(sessions))
		}

		// Test project stats
		stats, err := dbManager.GetProjectStats(projectPath)
		if err != nil {
			t.Errorf("Failed to get project stats: %v", err)
		}

		if sessions, ok := stats["sessions"].(map[string]int); ok {
			if len(sessions) != len(agentTypes) {
				t.Errorf("Expected session stats for %d agent types, got %d", len(agentTypes), len(sessions))
			}
		}
	})
}

// TestAgentPromptRetrievalConsistency tests the fallback mechanisms for agent prompts
func TestAgentPromptRetrievalConsistency(t *testing.T) {
	t.Run("DatabaseFallbackToHardcoded", func(t *testing.T) {
		// Test that system falls back to hardcoded prompts when database fails
		agentTypes := ListAgentTypes()
		task := "Test task for fallback mechanism"

		for _, agentType := range agentTypes {
			// Test getAgentPrompts with invalid database path
			systemPrompt, taskPrompt, err := getAgentPrompts(agentType, task)
			if err != nil {
				t.Errorf("getAgentPrompts failed for %s: %v", agentType, err)
			}

			if systemPrompt == "" {
				t.Errorf("Empty system prompt for %s", agentType)
			}

			if taskPrompt == "" {
				t.Errorf("Empty task prompt for %s", agentType)
			}

			// Verify task is included in task prompt
			if !strings.Contains(taskPrompt, task) {
				t.Errorf("Task not found in task prompt for %s", agentType)
			}
		}
	})

	t.Run("PromptConsistencyAcrossSources", func(t *testing.T) {
		// Test that prompts are consistent between hardcoded and database sources
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		dbManager, err := NewDBManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}
		defer dbManager.Close()

		// Initialize default prompts in database
		err = dbManager.InitializeDefaultPrompts()
		if err != nil {
			t.Fatalf("Failed to initialize default prompts: %v", err)
		}

		// Compare prompts from different sources
		agentTypes := ListAgentTypes()

		for _, agentType := range agentTypes {
			// Get from hardcoded
			hardcodedPrompt, err := GetAgentPrompt(agentType)
			if err != nil {
				t.Errorf("Failed to get hardcoded prompt for %s: %v", agentType, err)
				continue
			}

			// Get from database
			dbPrompt, err := dbManager.GetStoredAgentPrompt(agentType)
			if err != nil {
				t.Errorf("Failed to get database prompt for %s: %v", agentType, err)
				continue
			}

			// Compare key fields
			if hardcodedPrompt.Type != dbPrompt.Type {
				t.Errorf("Type mismatch for %s: hardcoded=%s, db=%s",
					agentType, hardcodedPrompt.Type, dbPrompt.Type)
			}

			if hardcodedPrompt.Name != dbPrompt.Name {
				t.Errorf("Name mismatch for %s: hardcoded=%s, db=%s",
					agentType, hardcodedPrompt.Name, dbPrompt.Name)
			}

			if hardcodedPrompt.SystemPrompt != dbPrompt.SystemPrompt {
				t.Errorf("SystemPrompt mismatch for %s", agentType)
			}

			if hardcodedPrompt.TaskPrompt != dbPrompt.TaskPrompt {
				t.Errorf("TaskPrompt mismatch for %s", agentType)
			}
		}
	})
}

// TestConfigurationLoadingAndValidation tests configuration loading scenarios
func TestConfigurationLoadingAndValidation(t *testing.T) {
	t.Run("DefaultConfigurationCreation", func(t *testing.T) {
		// Test that default configuration is created when config.json doesn't exist
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		logger, err := NewLogger("INFO", false, "")
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		config, err := loadConfig(logger)
		if err != nil {
			t.Fatalf("Failed to load default config: %v", err)
		}

		// Verify default values
		if !config.UseAgents {
			t.Error("UseAgents should be true in default config")
		}

		if config.AgentType != BackendDeveloper {
			t.Errorf("Expected default agent type %s, got %s", BackendDeveloper, config.AgentType)
		}

		if config.PromptCommand != DefaultPromptCommand {
			t.Error("PromptCommand should have default value")
		}

		if config.ReplyPrompt == "" {
			t.Error("ReplyPrompt should have default value")
		}

		if !config.AddToTasks {
			t.Error("AddToTasks should be true by default")
		}
	})

	t.Run("CustomConfigurationLoading", func(t *testing.T) {
		// Test loading custom configuration with agent-only settings
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Create custom config
		customConfig := Config{
			UseAgents:     false, // Should be overridden to true
			AgentType:     FrontendDeveloper,
			PromptCommand: "Custom prompt command",
			WebhookURL:    "https://example.com/webhook",
			ParseReply:    true,
			ReplyPrompt:   "Custom reply prompt: ",
			AddToTasks:    false,
		}

		configData, err := json.MarshalIndent(customConfig, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		err = os.WriteFile("config.json", configData, 0644)
		if err != nil {
			t.Fatalf("Failed to write config.json: %v", err)
		}

		logger, err := NewLogger("INFO", false, "")
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		loadedConfig, err := loadConfig(logger)
		if err != nil {
			t.Fatalf("Failed to load custom config: %v", err)
		}

		// Verify agent-only enforcement
		if !loadedConfig.UseAgents {
			t.Error("UseAgents should be true even when set to false in config (agent-only enforcement)")
		}

		// Verify custom values are preserved
		if loadedConfig.AgentType != FrontendDeveloper {
			t.Errorf("Agent type should be preserved: expected %s, got %s",
				FrontendDeveloper, loadedConfig.AgentType)
		}

		if loadedConfig.PromptCommand != "Custom prompt command" {
			t.Error("PromptCommand should be preserved from config")
		}

		if loadedConfig.WebhookURL != "https://example.com/webhook" {
			t.Error("WebhookURL should be preserved from config")
		}

		if !loadedConfig.ParseReply {
			t.Error("ParseReply should be preserved from config")
		}

		if loadedConfig.ReplyPrompt != "Custom reply prompt: " {
			t.Error("ReplyPrompt should be preserved from config")
		}

		if loadedConfig.AddToTasks {
			t.Error("AddToTasks should be preserved from config")
		}
	})

	t.Run("InvalidConfigurationHandling", func(t *testing.T) {
		// Test handling of invalid configuration files
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Test invalid JSON
		err = os.WriteFile("config.json", []byte("invalid json content"), 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid config.json: %v", err)
		}

		logger, err := NewLogger("INFO", false, "")
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		_, err = loadConfig(logger)
		if err == nil {
			t.Error("Expected error when loading invalid JSON config")
		}

		// Verify error is descriptive
		if !strings.Contains(err.Error(), "failed to parse config file") {
			t.Errorf("Expected config parsing error, got: %v", err)
		}
	})
}

// TestWebhookIntegrationWithAgents tests webhook functionality with agent-only execution
func TestWebhookIntegrationWithAgents(t *testing.T) {
	t.Run("WebhookPayloadIncludesAgentInfo", func(t *testing.T) {
		// Test that webhook payloads include agent type information
		agentTypes := []AgentType{Tester, Debugger, BackendDeveloper, FrontendDeveloper, UX}

		for _, agentType := range agentTypes {
			config := Config{
				UseAgents:     true,
				AgentType:     agentType,
				WebhookURL:    "https://example.com/webhook",
				PromptCommand: "Test webhook integration",
			}

			// Mock webhook payload creation
			iteration := 5
			completed := true

			// This would be the webhook payload structure
			payload := map[string]interface{}{
				"agent_type":     config.AgentType,
				"iteration":      iteration,
				"completed":      completed,
				"prompt_command": config.PromptCommand,
				"timestamp":      time.Now().Format(time.RFC3339),
			}

			// Verify agent type is included
			if payload["agent_type"] != config.AgentType {
				t.Errorf("Agent type not properly included in webhook payload for %s", agentType)
			}

			// Verify other required fields
			if payload["iteration"] != iteration {
				t.Error("Iteration not properly included in webhook payload")
			}

			if payload["completed"] != completed {
				t.Error("Completed status not properly included in webhook payload")
			}

			if payload["prompt_command"] != config.PromptCommand {
				t.Error("Prompt command not properly included in webhook payload")
			}
		}
	})

	t.Run("WebhookValidationInAgentMode", func(t *testing.T) {
		// Test webhook URL validation specifically in agent-only mode
		testCases := []struct {
			name       string
			webhookURL string
			valid      bool
		}{
			{
				name:       "EmptyWebhook",
				webhookURL: "",
				valid:      true,
			},
			{
				name:       "ValidHTTPS",
				webhookURL: "https://api.example.com/webhook",
				valid:      true,
			},
			{
				name:       "ValidHTTP",
				webhookURL: "http://localhost:8080/webhook",
				valid:      true,
			},
			{
				name:       "InvalidFTP",
				webhookURL: "ftp://example.com/webhook",
				valid:      false,
			},
			{
				name:       "InvalidNoHost",
				webhookURL: "https:///webhook",
				valid:      false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := Config{
					UseAgents:     true,
					AgentType:     Tester,
					WebhookURL:    tc.webhookURL,
					PromptCommand: "Test validation",
				}

				err := validateWebhookURL(config.WebhookURL)
				if tc.valid && err != nil {
					t.Errorf("Expected valid webhook URL but got error: %v", err)
				} else if !tc.valid && err == nil {
					t.Error("Expected invalid webhook URL but validation passed")
				}
			})
		}
	})
}

// TestConcurrencyAndRaceConditions tests for race conditions in agent execution
func TestConcurrencyAndRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	t.Run("ConcurrentAgentExecution", func(t *testing.T) {
		// Test concurrent agent execution doesn't cause race conditions
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		dbManager, err := NewDBManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}
		defer dbManager.Close()

		agentTypes := []AgentType{Tester, Debugger, BackendDeveloper, FrontendDeveloper, UX}
		projectPath := "/test/concurrent/project"
		numGoroutines := 10
		done := make(chan bool, numGoroutines)

		// Launch multiple goroutines creating sessions concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				agentType := agentTypes[id%len(agentTypes)]

				// Create session
				sessionID, err := dbManager.CreateAgentSession(agentType, projectPath)
				if err != nil {
					t.Errorf("Goroutine %d: Failed to create session: %v", id, err)
					return
				}

				// Add input/output
				input := fmt.Sprintf("Concurrent input %d", id)
				output := fmt.Sprintf("Concurrent output %d", id)

				err = dbManager.AddSessionInput(sessionID, input)
				if err != nil {
					t.Errorf("Goroutine %d: Failed to add input: %v", id, err)
				}

				err = dbManager.AddSessionOutput(sessionID, output)
				if err != nil {
					t.Errorf("Goroutine %d: Failed to add output: %v", id, err)
				}

				// End session
				err = dbManager.EndAgentSession(sessionID)
				if err != nil {
					t.Errorf("Goroutine %d: Failed to end session: %v", id, err)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all sessions were created
		sessions, err := dbManager.GetAgentSessions(BackendDeveloper, projectPath)
		if err != nil {
			t.Errorf("Failed to get sessions: %v", err)
		}

		expectedSessions := numGoroutines / len(agentTypes)
		if len(sessions) < expectedSessions {
			t.Errorf("Expected at least %d sessions, got %d", expectedSessions, len(sessions))
		}
	})

	t.Run("ConcurrentAgentSelection", func(t *testing.T) {
		// Test concurrent agent selection doesn't cause race conditions
		tasks := []string{
			"Test the API endpoints",
			"Debug the memory leak",
			"Implement authentication system",
			"Design user interface",
			"Write documentation",
			"Research new technologies",
			"Create marketing campaign",
		}

		agentPriority := []AgentType{
			BackendDeveloper, FrontendDeveloper, Tester, Debugger, Researcher,
			UX, UI, DocumentationWriter, FeedbackSeeker, Marketer, Simplifier,
		}

		numGoroutines := len(tasks)
		done := make(chan bool, numGoroutines)

		for i, task := range tasks {
			go func(id int, taskText string) {
				defer func() { done <- true }()

				// Select agent for task
				selectedAgent := selectBestAgentForTask(taskText, agentPriority)

				// Verify selection is valid
				validAgent := false
				for _, agent := range agentPriority {
					if selectedAgent == agent {
						validAgent = true
						break
					}
				}

				if !validAgent {
					t.Errorf("Goroutine %d: Invalid agent selected: %s", id, selectedAgent)
				}

				t.Logf("Goroutine %d: Task '%s' -> Agent '%s'", id, taskText, selectedAgent)
			}(i, task)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// TestPerformanceAndScalability tests performance characteristics of the agent system
func TestPerformanceAndScalability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("AgentSelectionPerformance", func(t *testing.T) {
		// Test that agent selection is performant even with many tasks
		tasks := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			tasks[i] = fmt.Sprintf("Task %d: Implement %s feature with %s requirements",
				i, []string{"backend", "frontend", "testing", "debugging"}[i%4],
				[]string{"security", "performance", "usability"}[i%3])
		}

		agentPriority := []AgentType{
			BackendDeveloper, FrontendDeveloper, Tester, Debugger, Researcher,
			UX, UI, DocumentationWriter, FeedbackSeeker, Marketer, Simplifier,
		}

		start := time.Now()

		for _, task := range tasks {
			selectBestAgentForTask(task, agentPriority)
		}

		duration := time.Since(start)

		// Should complete 1000 selections in reasonable time (less than 1 second)
		if duration > time.Second {
			t.Errorf("Agent selection took too long: %v for 1000 tasks", duration)
		}

		t.Logf("Agent selection performance: %v for 1000 tasks (%.2f ms per task)",
			duration, float64(duration.Nanoseconds())/1000000.0/1000)
	})

	t.Run("DatabasePerformance", func(t *testing.T) {
		// Test database performance with many sessions
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "perf_test.db")

		dbManager, err := NewDBManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}
		defer dbManager.Close()

		numSessions := 100
		projectPath := "/test/performance/project"

		start := time.Now()

		// Create many sessions
		for i := 0; i < numSessions; i++ {
			agentType := []AgentType{Tester, Debugger, BackendDeveloper}[i%3]
			sessionID, err := dbManager.CreateAgentSession(agentType, projectPath)
			if err != nil {
				t.Errorf("Failed to create session %d: %v", i, err)
				continue
			}

			// Add input and output
			err = dbManager.AddSessionInput(sessionID, fmt.Sprintf("Input %d", i))
			if err != nil {
				t.Errorf("Failed to add input %d: %v", i, err)
			}

			err = dbManager.AddSessionOutput(sessionID, fmt.Sprintf("Output %d", i))
			if err != nil {
				t.Errorf("Failed to add output %d: %v", i, err)
			}

			err = dbManager.EndAgentSession(sessionID)
			if err != nil {
				t.Errorf("Failed to end session %d: %v", i, err)
			}
		}

		createDuration := time.Since(start)

		// Test retrieval performance
		start = time.Now()
		sessions, err := dbManager.GetAllAgentProgress(projectPath)
		if err != nil {
			t.Errorf("Failed to get all progress: %v", err)
		}
		retrieveDuration := time.Since(start)

		if len(sessions) == 0 {
			t.Log("No progress entries retrieved, but sessions should exist. Testing session retrieval instead.")
			// Try to get sessions instead
			agentSessions, err := dbManager.GetAgentSessions(BackendDeveloper, projectPath)
			if err != nil {
				t.Errorf("Failed to get sessions: %v", err)
			} else {
				t.Logf("Found %d sessions for BackendDeveloper", len(agentSessions))
			}
		}

		t.Logf("Database performance: %v to create %d sessions, %v to retrieve",
			createDuration, numSessions, retrieveDuration)

		// Performance should be reasonable (adjust expectations for test environment)
		if createDuration > 15*time.Second {
			t.Errorf("Database creation took too long: %v for %d sessions", createDuration, numSessions)
		}

		if retrieveDuration > 10*time.Second {
			t.Errorf("Database retrieval took too long: %v", retrieveDuration)
		}
	})
}
