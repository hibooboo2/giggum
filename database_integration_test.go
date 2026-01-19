package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDatabaseOperationsWithAgentExecution tests database operations specific to agent execution
func TestDatabaseOperationsWithAgentExecution(t *testing.T) {
	tempDir := t.TempDir()

	// Create test database
	dbPath := filepath.Join(tempDir, "test.db")
	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer dbManager.Close()

	t.Run("AgentSessionCreation", func(t *testing.T) {
		// Test creating agent sessions for different agent types
		agentTypes := []AgentType{Tester, Debugger, BackendDeveloper, UX}
		projectPath := "/test/project"

		for _, agentType := range agentTypes {
			t.Run(string(agentType), func(t *testing.T) {
				sessionID, err := dbManager.CreateAgentSession(agentType, projectPath)
				if err != nil {
					t.Errorf("Failed to create session for %s: %v", agentType, err)
					return
				}

				if sessionID <= 0 {
					t.Errorf("Invalid session ID for %s: %d", agentType, sessionID)
				}

				// Test ending the session
				err = dbManager.EndAgentSession(sessionID)
				if err != nil {
					t.Errorf("Failed to end session for %s: %v", agentType, err)
				}
			})
		}
	})

	t.Run("SessionInputOutput", func(t *testing.T) {
		// Test adding input and output to sessions
		sessionID, err := dbManager.CreateAgentSession(Tester, "/test/project")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Test adding input
		input := "Test input for agent execution"
		err = dbManager.AddSessionInput(sessionID, input)
		if err != nil {
			t.Errorf("Failed to add session input: %v", err)
		}

		// Test adding output
		output := "Test output from agent execution"
		err = dbManager.AddSessionOutput(sessionID, output)
		if err != nil {
			t.Errorf("Failed to add session output: %v", err)
		}

		// Clean up
		err = dbManager.EndAgentSession(sessionID)
		if err != nil {
			t.Errorf("Failed to end session: %v", err)
		}
	})

	t.Run("MultipleSessionsForSameAgent", func(t *testing.T) {
		// Test creating multiple sessions for the same agent type
		projectPath := "/test/multiple/sessions"

		sessionIDs := make([]int64, 3)
		for j := 0; j < 3; j++ {
			sessionID, err := dbManager.CreateAgentSession(BackendDeveloper, projectPath)
			if err != nil {
				t.Errorf("Failed to create session %d: %v", j, err)
				continue
			}
			sessionIDs[j] = sessionID
		}

		// End all sessions
		for i, sessionID := range sessionIDs {
			if sessionID > 0 {
				err := dbManager.EndAgentSession(sessionID)
				if err != nil {
					t.Errorf("Failed to end session %d: %v", i, err)
				}
			}
		}
	})

	t.Run("ProjectSpecificSessions", func(t *testing.T) {
		// Test sessions for different projects
		projects := []string{"/project/a", "/project/b", "/project/c"}

		for _, projectPath := range projects {
			sessionID, err := dbManager.CreateAgentSession(Tester, projectPath)
			if err != nil {
				t.Errorf("Failed to create session for project %s: %v", projectPath, err)
				continue
			}

			// Add project-specific input
			input := "Test input for " + projectPath
			err = dbManager.AddSessionInput(sessionID, input)
			if err != nil {
				t.Errorf("Failed to add input for project %s: %v", projectPath, err)
			}

			err = dbManager.EndAgentSession(sessionID)
			if err != nil {
				t.Errorf("Failed to end session for project %s: %v", projectPath, err)
			}
		}
	})
}

// TestAgentPromptStorage tests storing and retrieving agent prompts in the database
func TestAgentPromptStorage(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer dbManager.Close()

	t.Run("StoreAndRetrievePrompt", func(t *testing.T) {
		agentType := Tester
		systemPrompt := "Test system prompt for tester agent"
		taskPrompt := "Test task prompt for tester agent"

		// Store the prompt
		prompt := AgentPrompt{
			Type:         agentType,
			Name:         string(agentType),
			Description:  "Test prompt for " + string(agentType),
			SystemPrompt: systemPrompt,
			TaskPrompt:   taskPrompt,
		}
		err := dbManager.StoreAgentPrompt(prompt)
		if err != nil {
			t.Fatalf("Failed to store agent prompt: %v", err)
		}

		// Retrieve the prompt
		storedPrompt, err := dbManager.GetStoredAgentPrompt(agentType)
		if err != nil {
			t.Fatalf("Failed to retrieve stored agent prompt: %v", err)
		}

		// Verify the stored prompt matches
		if storedPrompt.SystemPrompt != systemPrompt {
			t.Errorf("System prompt mismatch. Expected: %s, Got: %s", systemPrompt, storedPrompt.SystemPrompt)
		}

		if storedPrompt.TaskPrompt != taskPrompt {
			t.Errorf("Task prompt mismatch. Expected: %s, Got: %s", taskPrompt, storedPrompt.TaskPrompt)
		}
	})

	t.Run("MultipleAgentPrompts", func(t *testing.T) {
		// Test storing prompts for multiple agent types
		agentTypes := []AgentType{Tester, Debugger, BackendDeveloper}

		for _, agentType := range agentTypes {
			systemPrompt := "System prompt for " + string(agentType)
			taskPrompt := "Task prompt for " + string(agentType)

			prompt := AgentPrompt{
				Type:         agentType,
				Name:         string(agentType),
				Description:  "Test prompt for " + string(agentType),
				SystemPrompt: systemPrompt,
				TaskPrompt:   taskPrompt,
			}
			err := dbManager.StoreAgentPrompt(prompt)
			if err != nil {
				t.Errorf("Failed to store prompt for %s: %v", agentType, err)
			}
		}

		// Retrieve and verify all stored prompts
		for _, agentType := range agentTypes {
			storedPrompt, err := dbManager.GetStoredAgentPrompt(agentType)
			if err != nil {
				t.Errorf("Failed to retrieve prompt for %s: %v", agentType, err)
				continue
			}

			expectedSystem := "System prompt for " + string(agentType)
			expectedTask := "Task prompt for " + string(agentType)

			if storedPrompt.SystemPrompt != expectedSystem {
				t.Errorf("System prompt mismatch for %s. Expected: %s, Got: %s",
					agentType, expectedSystem, storedPrompt.SystemPrompt)
			}

			if storedPrompt.TaskPrompt != expectedTask {
				t.Errorf("Task prompt mismatch for %s. Expected: %s, Got: %s",
					agentType, expectedTask, storedPrompt.TaskPrompt)
			}
		}
	})

	t.Run("NonExistentPrompt", func(t *testing.T) {
		// Test retrieving a prompt that doesn't exist
		nonExistentAgent := AgentType("nonexistent")

		_, err := dbManager.GetStoredAgentPrompt(nonExistentAgent)
		if err == nil {
			t.Error("Expected error when retrieving non-existent prompt")
		}
	})
}

// TestDatabaseConcurrency tests concurrent database operations
func TestDatabaseConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent.db")

	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer dbManager.Close()

	t.Run("ConcurrentSessionCreation", func(t *testing.T) {
		// Test creating multiple sessions concurrently
		numSessions := 10
		done := make(chan bool, numSessions)
		sessionIDs := make([]int64, numSessions)

		for j := 0; j < numSessions; j++ {
			go func(index int) {
				defer func() { done <- true }()

				sessionID, err := dbManager.CreateAgentSession(Tester, "/concurrent/test")
				if err != nil {
					t.Errorf("Concurrent session creation failed for index %d: %v", index, err)
					return
				}
				sessionIDs[index] = sessionID
			}(j)
		}

		// Wait for all goroutines to complete
		for j := 0; j < numSessions; j++ {
			<-done
		}

		// End all created sessions
		for j, sessionID := range sessionIDs {
			if sessionID > 0 {
				err := dbManager.EndAgentSession(sessionID)
				if err != nil {
					t.Errorf("Failed to end concurrent session %d: %v", j, err)
				}
			}
		}
	})
}

// TestDatabaseIntegrityWithAgentExecution tests database integrity during agent execution scenarios
func TestDatabaseIntegrityWithAgentExecution(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integrity.db")

	// Create required files for agent execution simulation
	requiredFiles := []string{"tasks.md", "progress.txt", "prompt.md"}
	for _, file := range requiredFiles {
		err := os.WriteFile(filepath.Join(tempDir, file), []byte("# test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create %s: %v", file, err)
		}
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer dbManager.Close()

	t.Run("SessionLifecycleIntegrity", func(t *testing.T) {
		// Test complete session lifecycle
		agentType := BackendDeveloper
		projectPath := tempDir

		// Create session
		sessionID, err := dbManager.CreateAgentSession(agentType, projectPath)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add multiple inputs and outputs
		for j := 0; j < 3; j++ {
			input := "Input " + string(rune('A'+j))
			output := "Output " + string(rune('X'+j))

			err = dbManager.AddSessionInput(sessionID, input)
			if err != nil {
				t.Errorf("Failed to add input %d: %v", j, err)
			}

			err = dbManager.AddSessionOutput(sessionID, output)
			if err != nil {
				t.Errorf("Failed to add output %d: %v", j, err)
			}
		}

		// End session
		err = dbManager.EndAgentSession(sessionID)
		if err != nil {
			t.Errorf("Failed to end session: %v", err)
		}
	})

	t.Run("AgentTypeConsistency", func(t *testing.T) {
		// Test that agent types are consistent across operations
		agentTypes := ListAgentTypes()

		for _, agentType := range agentTypes {
			sessionID, err := dbManager.CreateAgentSession(agentType, tempDir)
			if err != nil {
				t.Errorf("Failed to create session for %s: %v", agentType, err)
				continue
			}

			// Store and retrieve prompt
			systemPrompt := "System for " + string(agentType)
			taskPrompt := "Task for " + string(agentType)

			prompt := AgentPrompt{
				Type:         agentType,
				Name:         string(agentType),
				Description:  "Test prompt for " + string(agentType),
				SystemPrompt: systemPrompt,
				TaskPrompt:   taskPrompt,
			}
			err = dbManager.StoreAgentPrompt(prompt)
			if err != nil {
				t.Errorf("Failed to store prompt for %s: %v", agentType, err)
			}

			retrievedPrompt, err := dbManager.GetStoredAgentPrompt(agentType)
			if err != nil {
				t.Errorf("Failed to retrieve prompt for %s: %v", agentType, err)
			}

			if retrievedPrompt.SystemPrompt != systemPrompt {
				t.Errorf("Prompt inconsistency for %s", agentType)
			}

			err = dbManager.EndAgentSession(sessionID)
			if err != nil {
				t.Errorf("Failed to end session for %s: %v", agentType, err)
			}
		}
	})
}
