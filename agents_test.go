package main

import (
	"testing"
)

func TestGetAgentPrompts(t *testing.T) {
	prompts := GetAgentPrompts()

	// Test that all expected agent types are present
	expectedTypes := ListAgentTypes()

	for _, agentType := range expectedTypes {
		prompt, exists := prompts[agentType]
		if !exists {
			t.Errorf("Agent type %s not found in prompts", agentType)
		}

		// Validate prompt structure
		if prompt.Type != agentType {
			t.Errorf("Prompt type mismatch for %s: expected %s, got %s", agentType, agentType, prompt.Type)
		}

		if prompt.Name == "" {
			t.Errorf("Empty name for agent type %s", agentType)
		}

		if prompt.Description == "" {
			t.Errorf("Empty description for agent type %s", agentType)
		}

		if prompt.SystemPrompt == "" {
			t.Errorf("Empty system prompt for agent type %s", agentType)
		}

		if prompt.TaskPrompt == "" {
			t.Errorf("Empty task prompt for agent type %s", agentType)
		}
	}
}

func TestGetAgentPrompt(t *testing.T) {
	// Test valid agent type
	prompt, err := GetAgentPrompt(Tester)
	if err != nil {
		t.Fatalf("Failed to get tester prompt: %v", err)
	}

	if prompt.Type != Tester {
		t.Errorf("Expected agent type %s, got %s", Tester, prompt.Type)
	}

	// Test invalid agent type
	invalidType := AgentType("invalid")
	_, err = GetAgentPrompt(invalidType)
	if err == nil {
		t.Error("Expected error for invalid agent type")
	}
}

func TestListAgentTypes(t *testing.T) {
	types := ListAgentTypes()

	expectedCount := 11 // Based on the constants defined
	if len(types) != expectedCount {
		t.Errorf("Expected %d agent types, got %d", expectedCount, len(types))
	}

	// Test that all expected types are present
	expectedTypes := []AgentType{
		Tester, Debugger, Researcher, BackendDeveloper, FrontendDeveloper,
		UX, UI, Marketer, FeedbackSeeker, Simplifier, DocumentationWriter,
	}

	for _, expectedType := range expectedTypes {
		found := false
		for _, actualType := range types {
			if actualType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected agent type %s not found in list", expectedType)
		}
	}
}

func TestCreateAgentPrompt(t *testing.T) {
	task := "Test this functionality"

	// Test all agent types
	agentTypes := ListAgentTypes()
	for _, agentType := range agentTypes {
		prompt, err := createAgentPrompt(agentType, task)
		if err != nil {
			t.Errorf("Failed to create prompt for %s: %v", agentType, err)
		}

		// Check that the task is included in the prompt
		if prompt == "" {
			t.Errorf("Empty prompt created for %s", agentType)
		}

		// The task should be mentioned in the prompt
		if !contains(prompt, task) {
			t.Errorf("Task not found in prompt for %s", agentType)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(hasPrefix(s, substr) || hasSuffix(s, substr) || containsInner(s, substr)))
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
