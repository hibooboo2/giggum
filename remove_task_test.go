package main

import (
	"path/filepath"
	"testing"
)

func TestRemoveTask(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create task manager
	taskManager, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer taskManager.Close()

	// Create a test task
	task, err := taskManager.CreateTask("Test Task", "This is a test task", "medium")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Verify task exists
	_, err = taskManager.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve created task: %v", err)
	}

	// Delete the task
	err = taskManager.DeleteTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify task is gone
	_, err = taskManager.GetTask(task.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted task, but got none")
	}
}
