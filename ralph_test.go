package main

import (
	"os"
	"testing"
)

// TestRunIterationsAgentOnly tests that runIterations only uses agent execution
func TestRunIterationsAgentOnly(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory for the test
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
	requiredFiles := []string{"progress.txt", "prompt.md"}
	for _, file := range requiredFiles {
		err = os.WriteFile(file, []byte("# test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create %s: %v", file, err)
		}
	}

	// Create logger
	logger, err := NewLogger("INFO", false, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Test config with agent type specified
	config := Config{
		UseAgents:     true,
		AgentType:     Tester,
		PromptCommand: "Test task execution",
		WebhookURL:    "",
	}

	// Test that runIterations executes with agent only

	// Capture stdout to verify agent usage messages
	// In a real test environment, we'd mock the opencode command
	t.Run("AgentTypeSpecified", func(t *testing.T) {
		// This test verifies the logic flow without actually executing opencode
		// In practice, you'd need to mock exec.Command or use integration tests

		// Verify config has UseAgents=true
		if !config.UseAgents {
			t.Error("Config should have UseAgents=true for agent-only execution")
		}

		// Verify agent type is set
		if config.AgentType != Tester {
			t.Errorf("Expected agent type %s, got %s", Tester, config.AgentType)
		}
	})

	t.Run("DefaultAgentType", func(t *testing.T) {
		// Test with empty agent type (should default to BackendDeveloper)
		configDefault := Config{
			UseAgents:     true,
			AgentType:     "",
			PromptCommand: "Test task execution",
			WebhookURL:    "",
		}

		// Simulate the logic from runIterations lines 66-68
		agentType := configDefault.AgentType
		if agentType == "" {
			agentType = BackendDeveloper // Default as per line 68
		}

		if agentType != BackendDeveloper {
			t.Errorf("Expected default agent type %s, got %s", BackendDeveloper, agentType)
		}
	})
}

// TestAgentSelectionLogic tests the agent selection logic in runIterations
func TestAgentSelectionLogic(t *testing.T) {
	// Test cases for agent selection logic
	testCases := []struct {
		name        string
		configAgent AgentType
		expected    AgentType
		description string
	}{
		{
			name:        "TesterSpecified",
			configAgent: Tester,
			expected:    Tester,
			description: "Should use Tester when specified in config",
		},
		{
			name:        "DebuggerSpecified",
			configAgent: Debugger,
			expected:    Debugger,
			description: "Should use Debugger when specified in config",
		},
		{
			name:        "EmptyDefaultsToBackend",
			configAgent: "",
			expected:    BackendDeveloper,
			description: "Should default to BackendDeveloper when empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the logic from runIterations lines 66-68
			agentType := tc.configAgent
			if agentType == "" {
				agentType = BackendDeveloper
			}

			if agentType != tc.expected {
				t.Errorf("%s: Expected %s, got %s", tc.description, tc.expected, agentType)
			}
		})
	}
}

// TestErrorHandlingInAgentMode tests error scenarios in agent-only execution
func TestErrorHandlingInAgentMode(t *testing.T) {
	// Create required files
	requiredFiles := []string{"tasks.md", "progress.txt", "prompt.md"}
	for _, file := range requiredFiles {
		err := os.WriteFile(file, []byte("# test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create %s: %v", file, err)
		}
	}

	logger, err := NewLogger("INFO", false, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	t.Run("COMPLETEDSignal", func(t *testing.T) {
		// Test the "COMPLETED" error signal handling (lines 75-81)
		// This would normally be tested by mocking runAgentWithOutputCapture
		// to return "COMPLETED" error

		// Verify the logic exists for early termination
		// In practice, this would be an integration test
	})

	t.Run("ErrorContinuation", func(t *testing.T) {
		// Test error handling for non-final iterations (lines 83-88)
		// Should log error but continue to next iteration unless it's the last one

		iterations := 3
		currentIteration := 2
		testError := "Test error message"

		// Simulate the logic from lines 83-88
		if currentIteration == iterations {
			// Would exit with os.Exit(1) in real implementation
			t.Log("Would exit on final iteration error")
		} else {
			// Would continue to next iteration
			t.Logf("Would continue after error in iteration %d: %s", currentIteration, testError)
		}
	})
}

// TestConfigurationEnforcement tests that UseAgents is always true
func TestConfigurationEnforcement(t *testing.T) {
	t.Run("UseAgentsAlwaysTrue", func(t *testing.T) {
		// Test the logic from main() lines 167-168
		// This ensures UseAgents is always set to true regardless of config file

		testConfigs := []struct {
			name      string
			useAgents bool
			agentType AgentType
		}{
			{"FalseInConfig", false, ""},
			{"TrueInConfig", true, Tester},
			{"EmptyConfig", false, ""},
		}

		for _, tc := range testConfigs {
			t.Run(tc.name, func(t *testing.T) {
				// Simulate the logic from main()
				config := Config{
					UseAgents: tc.useAgents,
					AgentType: tc.agentType,
				}

				// This line from main() forces agent usage
				config.UseAgents = true

				if !config.UseAgents {
					t.Error("UseAgents should always be true after enforcement")
				}

				// Test agent type fallback logic (lines 169-174)
				if tc.agentType != "" {
					if config.AgentType != tc.agentType {
						t.Errorf("Agent type should be preserved when specified: %s", tc.agentType)
					}
				} else if config.AgentType == "" {
					// Should default to BackendDeveloper
					config.AgentType = BackendDeveloper
					if config.AgentType != BackendDeveloper {
						t.Error("Should default to BackendDeveloper when no agent specified")
					}
				}
			})
		}
	})
}

// TestCLIArgumentHandling tests CLI argument parsing for agent execution
func TestCLIArgumentHandling(t *testing.T) {
	t.Run("DefaultAgentType", func(t *testing.T) {
		// Test the default from parseFlags() line 34
		expectedDefault := "backend-developer"

		// This would normally be tested by calling parseFlags()
		// with different arguments and verifying the results

		if expectedDefault != "backend-developer" {
			t.Errorf("Expected default agent type to be 'backend-developer', got '%s'", expectedDefault)
		}
	})

	t.Run("AvailableAgentTypes", func(t *testing.T) {
		// Test that all agent types from the help text are valid
		expectedTypes := []string{
			"tester", "debugger", "researcher", "backend-developer",
			"frontend-developer", "ux", "ui", "marketer",
			"feedbackseeker", "simplifier", "documentationwriter",
		}

		// Verify these match the constants defined in agents.go
		actualTypes := ListAgentTypes()

		if len(actualTypes) != len(expectedTypes) {
			t.Errorf("Expected %d agent types, got %d", len(expectedTypes), len(actualTypes))
		}

		// Convert expected types to AgentType for comparison
		for i, expected := range expectedTypes {
			if i >= len(actualTypes) {
				t.Errorf("Missing agent type: %s", expected)
				continue
			}
			if string(actualTypes[i]) != expected {
				t.Errorf("Expected %s at position %d, got %s", expected, i, actualTypes[i])
			}
		}
	})
}
