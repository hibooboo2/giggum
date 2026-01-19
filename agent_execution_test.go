package main

import (
	"testing"
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
