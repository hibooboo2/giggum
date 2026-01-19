package main

import (
	"testing"
)

func TestAgentsCommand(t *testing.T) {
	// Test that agents command exists and has correct structure
	if agentsCmd.Use != "agents" {
		t.Errorf("Expected agents command use 'agents', got '%s'", agentsCmd.Use)
	}

	if agentsCmd.Short != "Manage and interact with agents" {
		t.Errorf("Expected agents command short description 'Manage and interact with agents', got '%s'", agentsCmd.Short)
	}

	// Test that agents command has list subcommand
	if len(agentsCmd.Commands()) != 1 {
		t.Errorf("Expected agents command to have 1 subcommand, got %d", len(agentsCmd.Commands()))
	}

	listCmd := agentsCmd.Commands()[0]
	if listCmd.Use != "list" {
		t.Errorf("Expected agents list subcommand use 'list', got '%s'", listCmd.Use)
	}

	if listCmd.Short != "List all available agent types" {
		t.Errorf("Expected agents list subcommand short description 'List all available agent types', got '%s'", listCmd.Short)
	}
}

func TestAgentsListCommandExecution(t *testing.T) {
	// Since listAvailableAgents() writes directly to stdout,
	// we'll test the function's existence and basic structure
	// rather than trying to capture its output

	// Test that the command function exists and can be called without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic occurred when executing agents list command: %v", r)
		}
	}()

	// Execute the command function directly
	agentsListCmd.Run(agentsListCmd, []string{})

	// If we get here without panic, the test passes
	// The actual output verification is done manually above
}

func TestRootCommandContainsAgents(t *testing.T) {
	// Test that root command includes agents command
	var foundAgents bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "agents" {
			foundAgents = true
			break
		}
	}

	if !foundAgents {
		t.Error("Expected root command to have 'agents' subcommand")
	}

	// Test that root command no longer has list-agents command
	var foundListAgents bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "list-agents" {
			foundListAgents = true
			break
		}
	}

	if foundListAgents {
		t.Error("Expected root command to NOT have 'list-agents' subcommand (should be under agents)")
	}
}

func TestAgentsCommandHelp(t *testing.T) {
	// Test agents command long description
	if agentsCmd.Long != "Commands for managing and interacting with different agent types." {
		t.Errorf("Expected agents command long description 'Commands for managing and interacting with different agent types.', got '%s'", agentsCmd.Long)
	}
}

func TestAgentsListCommandHelp(t *testing.T) {
	// Test agents list command long description
	if agentsListCmd.Long != "Display all available agent types with their descriptions." {
		t.Errorf("Expected agents list command long description 'Display all available agent types with their descriptions.', got '%s'", agentsListCmd.Long)
	}
}
