package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTasksFile(t *testing.T) {
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

	// Test reading non-existent file
	_, err = dbManager.ReadTasksFile(projectPath)
	if err == nil {
		t.Error("Expected error when reading non-existent tasks.md file")
	}

	// Create the project directory
	err = os.MkdirAll(projectPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create tasks.md file
	tasksContent := "# Test Tasks\n\n- Task 1\n- Task 2\n"
	tasksFile := filepath.Join(projectPath, "tasks.md")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tasks.md file: %v", err)
	}

	// Test reading existing file
	content, err := dbManager.ReadTasksFile(projectPath)
	if err != nil {
		t.Errorf("Failed to read tasks.md file: %v", err)
	}

	if content != tasksContent {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", tasksContent, content)
	}
}

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

func TestAddTaskToProject(t *testing.T) {
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
	err = os.MkdirAll(projectPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Test adding task to non-existent file (should create new file)
	err = dbManager.AddTaskToProject(projectPath, "New test task", "")
	if err != nil {
		t.Errorf("Failed to add task to new project: %v", err)
	}

	// Verify file was created
	tasksFile := filepath.Join(projectPath, "tasks.md")
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		t.Error("Tasks.md file was not created")
	}

	// Read content
	content, err := os.ReadFile(tasksFile)
	if err != nil {
		t.Errorf("Failed to read tasks.md file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "New test task") {
		t.Errorf("Task was not added to file. Content: %s", contentStr)
	}

	// Test adding another task with priority
	err = dbManager.AddTaskToProject(projectPath, "Another task", "low")
	if err != nil {
		t.Errorf("Failed to add task with priority: %v", err)
	}

	// Read content again
	content, err = os.ReadFile(tasksFile)
	if err != nil {
		t.Errorf("Failed to read tasks.md file: %v", err)
	}

	contentStr = string(content)
	if !contains(contentStr, "Another task (low)") {
		t.Errorf("Task with priority was not added to file. Content: %s", contentStr)
	}
}
