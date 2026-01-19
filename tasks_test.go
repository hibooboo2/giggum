package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTasksFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create database manager
	dbManager, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create DB manager: %v", err)
	}
	defer dbManager.Close()

	// Create a test project directory
	projectPath := filepath.Join(tempDir, "test-project")

	// Test writing to non-existent directory
	content := "# Test Tasks\n\n- Task 1\n"
	err = dbManager.WriteTasksFile(projectPath, content)
	if err != nil {
		t.Errorf("Failed to write tasks.md file: %v", err)
	}

	// Verify file was created
	tasksFile := filepath.Join(projectPath, "tasks.md")
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		t.Error("Tasks.md file was not created")
	}

	// Verify content
	readContent, err := os.ReadFile(tasksFile)
	if err != nil {
		t.Errorf("Failed to read tasks.md file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", content, string(readContent))
	}
}

func TestTaskManager(t *testing.T) {
	// Use a temporary database for testing
	testDB := "./test_tasks.db"
	defer os.Remove(testDB)

	// Create task manager
	tm, err := NewTaskManager(testDB)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test creating a task
	task, err := tm.CreateTask("Test Task", "This is a test task", "high")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if task.Title != "Test Task" {
		t.Errorf("Expected task title 'Test Task', got '%s'", task.Title)
	}

	if task.Priority != "high" {
		t.Errorf("Expected task priority 'high', got '%s'", task.Priority)
	}

	if task.Status != "pending" {
		t.Errorf("Expected task status 'pending', got '%s'", task.Status)
	}

	// Test getting task by ID
	retrievedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrievedTask.Title != task.Title {
		t.Errorf("Retrieved task title mismatch. Expected '%s', got '%s'", task.Title, retrievedTask.Title)
	}

	// Test updating task status
	err = tm.UpdateTaskStatus(task.ID, "completed")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	updatedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updatedTask.Status != "completed" {
		t.Errorf("Expected updated status 'completed', got '%s'", updatedTask.Status)
	}

	if updatedTask.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set when status is completed")
	}

	// Test getting all tasks
	allTasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(allTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(allTasks))
	}

	// Test task stats
	stats, err := tm.GetTaskStats()
	if err != nil {
		t.Fatalf("Failed to get task stats: %v", err)
	}

	if stats["total"] != 1 {
		t.Errorf("Expected total stats 1, got %d", stats["total"])
	}

	if stats["completed"] != 1 {
		t.Errorf("Expected completed stats 1, got %d", stats["completed"])
	}

	// Test deleting task
	err = tm.DeleteTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	allTasksAfterDelete, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks after delete: %v", err)
	}

	if len(allTasksAfterDelete) != 0 {
		t.Errorf("Expected 0 tasks after delete, got %d", len(allTasksAfterDelete))
	}
}

func TestTaskManagerMarkdownImport(t *testing.T) {
	// Use a temporary database for testing
	testDB := "./test_import_tasks.db"
	defer os.Remove(testDB)

	// Create task manager
	tm, err := NewTaskManager(testDB)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test markdown import
	markdownContent := `# Project Tasks

## HIGH PRIORITY
- Critical security fix (high)
- Database migration (high)

## Medium Priority
- Add user authentication (medium)
- Update documentation (medium)

## LOW PRIORITY
- Improve UI colors (low)`

	err = tm.ImportTasksFromMarkdown(markdownContent)
	if err != nil {
		t.Fatalf("Failed to import from markdown: %v", err)
	}

	// Verify tasks were imported
	tasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	expectedCount := 5 // 2 high + 2 medium + 1 low
	if len(tasks) != expectedCount {
		t.Errorf("Expected %d tasks, got %d", expectedCount, len(tasks))
	}

	// Check priority distribution
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, task := range tasks {
		switch task.Priority {
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
	}

	if highCount != 2 {
		t.Errorf("Expected 2 high priority tasks, got %d", highCount)
	}
	if mediumCount != 2 {
		t.Errorf("Expected 2 medium priority tasks, got %d", mediumCount)
	}
	if lowCount != 1 {
		t.Errorf("Expected 1 low priority task, got %d", lowCount)
	}
}
