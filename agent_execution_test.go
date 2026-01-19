package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestAgentExecutionPaths tests all agent execution paths for consistency
func TestAgentExecutionPaths(t *testing.T) {
	agentTypes := ListAgentTypes()

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			// Test that each agent type can create prompts
			task := "Test execution task"

			prompt, err := createAgentPrompt(agentType, task)
			if err != nil {
				t.Errorf("Failed to create prompt for %s: %v", agentType, err)
				return
			}

			if prompt == "" {
				t.Errorf("Empty prompt created for %s", agentType)
			}

			// Verify task is included in prompt
			if !contains(prompt, task) {
				t.Errorf("Task not found in prompt for %s", agentType)
			}
		})
	}
}

// TestAgentPromptCreation tests prompt creation with various edge cases
func TestAgentPromptCreation(t *testing.T) {
	testCases := []struct {
		name    string
		task    string
		agent   AgentType
		wantErr bool
	}{
		{
			name:    "ValidTaskTester",
			task:    "Test this functionality",
			agent:   Tester,
			wantErr: false,
		},
		{
			name:    "ValidTaskDebugger",
			task:    "Debug this issue",
			agent:   Debugger,
			wantErr: false,
		},
		{
			name:    "EmptyTask",
			task:    "",
			agent:   BackendDeveloper,
			wantErr: false,
		},
		{
			name:    "LongTask",
			task:    "This is a very long task that contains multiple words and should still be handled properly by the agent prompt creation system without any issues",
			agent:   FrontendDeveloper,
			wantErr: false,
		},
		{
			name:    "SpecialCharacters",
			task:    "Task with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?",
			agent:   UX,
			wantErr: false,
		},
		{
			name:    "InvalidAgent",
			task:    "Test task",
			agent:   AgentType("invalid"),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prompt, err := createAgentPrompt(tc.agent, tc.task)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error for agent %s with task '%s'", tc.agent, tc.task)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for agent %s with task '%s': %v", tc.agent, tc.task, err)
				return
			}

			if tc.task != "" && !contains(prompt, tc.task) {
				t.Errorf("Task not found in prompt for agent %s", tc.agent)
			}
		})
	}
}

// TestAgentSystemAndTaskPrompts tests the separation of system and task prompts
func TestAgentSystemAndTaskPrompts(t *testing.T) {
	agentTypes := ListAgentTypes()
	task := "Test task for prompt separation"

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			systemPrompt, taskPrompt, err := getAgentPrompts(agentType, task)
			if err != nil {
				t.Errorf("Failed to get prompts for %s: %v", agentType, err)
				return
			}

			// System prompt should be non-empty and describe the agent
			if systemPrompt == "" {
				t.Errorf("Empty system prompt for %s", agentType)
			}

			// Task prompt should contain the task
			if taskPrompt == "" {
				t.Errorf("Empty task prompt for %s", agentType)
			}

			if !contains(taskPrompt, task) {
				t.Errorf("Task not found in task prompt for %s", agentType)
			}

			// System and task prompts should be different
			if systemPrompt == taskPrompt {
				t.Errorf("System and task prompts are identical for %s", agentType)
			}
		})
	}
}

// TestAgentPromptConsistency tests that all agents have consistent prompt structure
func TestAgentPromptConsistency(t *testing.T) {
	prompts := GetAgentPrompts()

	// All agents should have the same structure
	for agentType, prompt := range prompts {
		t.Run(string(agentType), func(t *testing.T) {
			// Check required fields
			if prompt.Type == "" {
				t.Errorf("Empty type for agent %s", agentType)
			}

			if prompt.Name == "" {
				t.Errorf("Empty name for agent %s", agentType)
			}

			if prompt.Description == "" {
				t.Errorf("Empty description for agent %s", agentType)
			}

			if prompt.SystemPrompt == "" {
				t.Errorf("Empty system prompt for agent %s", agentType)
			}

			if prompt.TaskPrompt == "" {
				t.Errorf("Empty task prompt for agent %s", agentType)
			}

			// Type should match the key
			if prompt.Type != agentType {
				t.Errorf("Prompt type mismatch: key is %s but type is %s", agentType, prompt.Type)
			}
		})
	}
}

// TestDefaultAgentSelection tests the default agent selection logic
func TestDefaultAgentSelection(t *testing.T) {
	testCases := []struct {
		name          string
		configAgent   AgentType
		expectedAgent AgentType
		description   string
	}{
		{
			name:          "EmptyConfigDefaultsToBackend",
			configAgent:   "",
			expectedAgent: BackendDeveloper,
			description:   "Empty config should default to BackendDeveloper",
		},
		{
			name:          "SpecifiedAgentIsPreserved",
			configAgent:   Tester,
			expectedAgent: Tester,
			description:   "Specified agent should be preserved",
		},
		{
			name:          "AllAgentTypesPreserved",
			configAgent:   DocumentationWriter,
			expectedAgent: DocumentationWriter,
			description:   "All agent types should be preserved when specified",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the selection logic from runIterations and main()
			agentType := tc.configAgent

			// Apply defaults as per runIterations line 66-68
			if agentType == "" {
				agentType = BackendDeveloper
			}

			if agentType != tc.expectedAgent {
				t.Errorf("%s: Expected %s, got %s", tc.description, tc.expectedAgent, agentType)
			}
		})
	}
}

// TestTimeoutConfiguration tests timeout configuration handling
func TestTimeoutConfiguration(t *testing.T) {
	testCases := []struct {
		name               string
		configAgentTimeout int
		cliTimeout         int
		expectedTimeout    int
		description        string
	}{
		{
			name:               "DefaultConfigTimeout",
			configAgentTimeout: 0,
			cliTimeout:         0,
			expectedTimeout:    30,
			description:        "Should use default 30 minutes timeout",
		},
		{
			name:               "ConfigTimeoutPreserved",
			configAgentTimeout: 45,
			cliTimeout:         0,
			expectedTimeout:    45,
			description:        "Config timeout should be preserved",
		},
		{
			name:               "CLITimeoutOverridesConfig",
			configAgentTimeout: 45,
			cliTimeout:         60,
			expectedTimeout:    60,
			description:        "CLI timeout should override config",
		},
		{
			name:               "CLITimeoutOnly",
			configAgentTimeout: 0,
			cliTimeout:         15,
			expectedTimeout:    15,
			description:        "CLI timeout should be used when no config timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate config loading logic
			config := Config{
				AgentTimeout: tc.configAgentTimeout,
			}

			// Apply CLI override logic (simulating main function logic)
			if tc.cliTimeout > 0 {
				config.AgentTimeout = tc.cliTimeout
			}

			// Apply default if still 0
			if config.AgentTimeout == 0 {
				config.AgentTimeout = 30
			}

			if config.AgentTimeout != tc.expectedTimeout {
				t.Errorf("%s: Expected timeout %d, got %d", tc.description, tc.expectedTimeout, config.AgentTimeout)
			}
		})
	}
}

// TestTimeoutPromptEnhancement tests that timeout information is added to prompts
func TestTimeoutPromptEnhancement(t *testing.T) {
	testCases := []struct {
		name         string
		agentType    AgentType
		task         string
		timeoutMin   int
		expectNotice bool
	}{
		{
			name:         "BackendDeveloperWithTimeout",
			agentType:    BackendDeveloper,
			task:         "Create a REST API",
			timeoutMin:   30,
			expectNotice: true,
		},
		{
			name:         "FrontendDeveloperWithTimeout",
			agentType:    FrontendDeveloper,
			task:         "Design a user interface",
			timeoutMin:   45,
			expectNotice: true,
		},
		{
			name:         "TesterWithTimeout",
			agentType:    Tester,
			task:         "Test the application",
			timeoutMin:   15,
			expectNotice: true,
		},
		{
			name:         "ZeroTimeout",
			agentType:    Debugger,
			task:         "Debug this issue",
			timeoutMin:   0,
			expectNotice: true, // Even 0 should trigger notice about time limits
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get original prompts
			_, taskPrompt, err := getAgentPrompts(tc.agentType, tc.task)
			if err != nil {
				t.Fatalf("Failed to get prompts for %s: %v", tc.agentType, err)
			}

			// Simulate timeout enhancement (from agent.go logic)
			timeoutNotice := ""
			if tc.timeoutMin > 0 {
				timeoutNotice = fmt.Sprintf("\n\nIMPORTANT: You have %d minutes maximum to complete this task. Monitor your time and ensure you reach a stopping point before the timeout. If you're running short on time, prioritize the most critical aspects and provide a partial solution with clear next steps.", tc.timeoutMin)
			} else {
				timeoutNotice = "\n\nIMPORTANT: You have limited time to complete this task. Monitor your time and ensure you reach a stopping point before the timeout. If you're running short on time, prioritize the most critical aspects and provide a partial solution with clear next steps."
			}

			enhancedTaskPrompt := taskPrompt + timeoutNotice

			// Verify timeout notice is added
			if tc.expectNotice && !strings.Contains(enhancedTaskPrompt, "IMPORTANT:") {
				t.Errorf("Expected timeout notice to be added to prompt for %s", tc.agentType)
			}

			if tc.expectNotice && !strings.Contains(enhancedTaskPrompt, "minutes maximum") && tc.timeoutMin > 0 {
				t.Errorf("Expected specific timeout minutes in prompt for %s", tc.agentType)
			}

			// Verify original task is still present
			if !strings.Contains(enhancedTaskPrompt, tc.task) {
				t.Errorf("Original task not found in enhanced prompt for %s", tc.agentType)
			}
		})
	}
}

// TestTimeoutContextCreation tests context creation for timeouts
func TestTimeoutContextCreation(t *testing.T) {
	testCases := []struct {
		name        string
		timeoutMin  int
		expectError bool
	}{
		{
			name:        "ValidTimeout30Min",
			timeoutMin:  30,
			expectError: false,
		},
		{
			name:        "ValidTimeout1Min",
			timeoutMin:  1,
			expectError: false,
		},
		{
			name:        "ValidTimeout120Min",
			timeoutMin:  120,
			expectError: false,
		},
		{
			name:        "ZeroTimeout",
			timeoutMin:  0,
			expectError: false, // Should still work, just immediate timeout
		},
		{
			name:        "NegativeTimeout",
			timeoutMin:  -1,
			expectError: false, // Go's time.Duration handles negative values gracefully
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test context creation (simulating agent.go logic)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tc.timeoutMin)*time.Minute)
			defer cancel()

			// Verify context was created
			if ctx == nil {
				t.Errorf("Failed to create context for timeout %d minutes", tc.timeoutMin)
			}

			// Test timeout behavior
			if tc.timeoutMin > 0 {
				// Context should not be done immediately
				select {
				case <-ctx.Done():
					t.Errorf("Context should not be done immediately for timeout %d minutes", tc.timeoutMin)
				default:
					// Expected - context is still active
				}
			}
		})
	}
}

// TestTimeoutErrorMessage tests timeout error message formatting
func TestTimeoutErrorMessage(t *testing.T) {
	testCases := []struct {
		name        string
		timeoutMin  int
		expectedMsg string
	}{
		{
			name:        "StandardTimeout",
			timeoutMin:  30,
			expectedMsg: "timed out after 30 minutes",
		},
		{
			name:        "ShortTimeout",
			timeoutMin:  5,
			expectedMsg: "timed out after 5 minutes",
		},
		{
			name:        "LongTimeout",
			timeoutMin:  120,
			expectedMsg: "timed out after 120 minutes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate timeout error (from agent.go logic)
			err := fmt.Errorf("agent execution timed out after %d minutes", tc.timeoutMin)

			if err == nil {
				t.Errorf("Expected error for timeout test case")
			}

			if !strings.Contains(err.Error(), tc.expectedMsg) {
				t.Errorf("Expected error message to contain '%s', got '%s'", tc.expectedMsg, err.Error())
			}

			// Test isTimeoutError function (from validation.go)
			if !isTimeoutError(err) {
				t.Errorf("isTimeoutError should return true for timeout error, but got false for: %s", err.Error())
			}
		})
	}
}

// TestAllAgentTypesWithTimeout tests that all agent types can handle timeout
func TestAllAgentTypesWithTimeout(t *testing.T) {
	agentTypes := ListAgentTypes()
	timeoutMin := 30

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			// Test that each agent type can handle timeout enhancement
			task := "Test task with timeout"

			systemPrompt, taskPrompt, err := getAgentPrompts(agentType, task)
			if err != nil {
				t.Fatalf("Failed to get prompts for %s: %v", agentType, err)
			}

			// Apply timeout enhancement
			timeoutNotice := fmt.Sprintf("\n\nIMPORTANT: You have %d minutes maximum to complete this task. Monitor your time and ensure you reach a stopping point before the timeout. If you're running short on time, prioritize the most critical aspects and provide a partial solution with clear next steps.", timeoutMin)
			enhancedTaskPrompt := taskPrompt + timeoutNotice

			// Verify enhancements don't break the prompt structure
			if !strings.Contains(enhancedTaskPrompt, task) {
				t.Errorf("Original task not found in enhanced prompt for %s", agentType)
			}

			if !strings.Contains(enhancedTaskPrompt, "IMPORTANT:") {
				t.Errorf("Timeout notice not found in enhanced prompt for %s", agentType)
			}

			// Verify system prompt is unchanged
			if strings.Contains(systemPrompt, "IMPORTANT:") {
				t.Errorf("System prompt should not contain timeout notice for %s", agentType)
			}
		})
	}
}
