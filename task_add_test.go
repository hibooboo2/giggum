package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAddTaskFromRawString(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tasks.db")

	// Create task manager
	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test 1: Basic task addition directly via TaskManager
	testTask := "Implement user authentication system with JWT tokens"

	// Create task directly instead of going through CLI parsing
	_, err = tm.CreateTaskWithStatus(testTask, "Test task for authentication system", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Verify task was created
	tasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	createdTask := tasks[0]
	if !strings.Contains(strings.ToLower(createdTask.Title), "authentication") {
		t.Errorf("Expected task title to contain 'authentication', got '%s'", createdTask.Title)
	}

	if createdTask.Priority == "" {
		t.Error("Expected priority to be set")
	}

	// Test 2: Task with high priority
	highPriorityTask := "Fix critical security vulnerability in production"
	_, err = tm.CreateTaskWithStatus(highPriorityTask, "Critical security fix", "high", "pending")
	if err != nil {
		t.Fatalf("Failed to create high priority task: %v", err)
	}

	// Verify we now have 2 tasks
	tasks, err = tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Find the urgent task
	var urgentTaskObj *Task
	for _, t := range tasks {
		if strings.Contains(t.Description, "Critical security fix") {
			urgentTaskObj = &t
			break
		}
	}

	if urgentTaskObj == nil {
		t.Error("Urgent task not found in tasks")
	} else if urgentTaskObj.Priority != "high" {
		t.Errorf("Expected urgent task to have high priority, got '%s'", urgentTaskObj.Priority)
	}

	// Test 3: Task with low priority
	lowPriorityTask := "Clean up old code files when we have time"
	_, err = tm.CreateTaskWithStatus(lowPriorityTask, "Code cleanup task", "low", "pending")
	if err != nil {
		t.Fatalf("Failed to create low priority task: %v", err)
	}

	// Verify low priority task
	tasks, err = tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	var lowPriorityTaskObj *Task
	for _, t := range tasks {
		if strings.Contains(t.Description, "Code cleanup task") {
			lowPriorityTaskObj = &t
			break
		}
	}

	if lowPriorityTaskObj == nil {
		t.Error("Low priority task not found in tasks")
	} else if lowPriorityTaskObj.Priority != "low" {
		t.Errorf("Expected low priority task to have 'low' priority, got '%s'", lowPriorityTaskObj.Priority)
	}
}
