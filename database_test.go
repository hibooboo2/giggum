package main

import (
	"path/filepath"
	"testing"
)

func TestNewDBManager(t *testing.T) {
	// Create a temporary database for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test successful creation
	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create DBManager: %v", err)
	}
	defer dbManager.Close()

	// Test database connection
	if dbManager.db == nil {
		t.Fatal("Database connection is nil")
	}
}

func TestAgentSessionLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create DBManager: %v", err)
	}
	defer dbManager.Close()

	// Test session creation
	sessionID, err := dbManager.CreateAgentSession(Tester, "/test/project")
	if err != nil {
		t.Fatalf("Failed to create agent session: %v", err)
	}
	if sessionID <= 0 {
		t.Fatal("Session ID should be positive")
	}

	// Test adding input
	err = dbManager.AddSessionInput(sessionID, "test input")
	if err != nil {
		t.Fatalf("Failed to add session input: %v", err)
	}

	// Test adding output
	err = dbManager.AddSessionOutput(sessionID, "test output")
	if err != nil {
		t.Fatalf("Failed to add session output: %v", err)
	}

	// Test ending session
	err = dbManager.EndAgentSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to end agent session: %v", err)
	}
}

func TestCreateAgentProgress(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create DBManager: %v", err)
	}
	defer dbManager.Close()

	// Test creating progress entry
	progressID, err := dbManager.CreateAgentProgress(Tester, "/test/project", "test task", 1)
	if err != nil {
		t.Fatalf("Failed to create agent progress: %v", err)
	}
	if progressID <= 0 {
		t.Fatal("Progress ID should be positive")
	}

	// Test updating progress
	err = dbManager.UpdateAgentProgress(progressID, "completed", "task completed successfully")
	if err != nil {
		t.Fatalf("Failed to update agent progress: %v", err)
	}
}

func TestGetAgentSessions(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create DBManager: %v", err)
	}
	defer dbManager.Close()

	// Create a test session
	sessionID, err := dbManager.CreateAgentSession(Tester, "/test/project")
	if err != nil {
		t.Fatalf("Failed to create agent session: %v", err)
	}

	// Add test data
	err = dbManager.AddSessionInput(sessionID, "test input")
	if err != nil {
		t.Fatalf("Failed to add session input: %v", err)
	}

	err = dbManager.AddSessionOutput(sessionID, "test output")
	if err != nil {
		t.Fatalf("Failed to add session output: %v", err)
	}

	err = dbManager.EndAgentSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to end agent session: %v", err)
	}

	// Retrieve sessions
	sessions, err := dbManager.GetAgentSessions(Tester, "/test/project")
	if err != nil {
		t.Fatalf("Failed to get agent sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	if session.AgentType != Tester {
		t.Errorf("Expected agent type %s, got %s", Tester, session.AgentType)
	}
	if len(session.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(session.Inputs))
	}
	if len(session.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(session.Outputs))
	}
}
