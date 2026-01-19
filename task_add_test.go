package main

import (
	"flag"
	"os"
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

	// Test 1: Basic task addition
	// Reset flag args for next test
	defer func() {
		flag.CommandLine = flag.NewFlagSet("", flag.ContinueOnError)
	}()

	// Simulate command line args for task add
	testTask := "Implement user authentication system with JWT tokens"
	os.Args = []string{"test", "-task", "add", testTask}

	// Parse flags (this sets flag.Args() correctly)
	_ = parseFlags()

	// Test the addTaskFromRawString function directly
	err = addTaskFromRawString(tm)
	if err != nil {
		t.Fatalf("Failed to add task from raw string: %v", err)
	}

	// Verify the task was created
	tasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if !strings.Contains(task.Title, "user authentication") {
		t.Errorf("Expected task title to contain 'user authentication', got '%s'", task.Title)
	}

	if task.Priority == "" {
		t.Error("Expected priority to be set")
	}

	// Test 2: Task with urgent keyword should get high priority
	urgentTask := "Fix critical security vulnerability in production"
	os.Args = []string{"test", "-task", "add", urgentTask}
	_ = parseFlags()

	err = addTaskFromRawString(tm)
	if err != nil {
		t.Fatalf("Failed to add urgent task: %v", err)
	}

	// Verify the urgent task was created with high priority
	tasks, err = tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Find the urgent task (should be the most recent)
	var urgentTaskObj *Task
	for _, t := range tasks {
		if strings.Contains(t.Description, "security vulnerability") {
			urgentTaskObj = &t
			break
		}
	}

	if urgentTaskObj == nil {
		t.Error("Urgent task not found in tasks")
	} else if urgentTaskObj.Priority != "high" {
		t.Errorf("Expected urgent task to have high priority, got '%s'", urgentTaskObj.Priority)
	}

	// Test 3: Task with database keyword should get database tags
	dbTask := "Optimize database queries for better performance"
	os.Args = []string{"test", "-task", "add", dbTask}
	_ = parseFlags()

	err = addTaskFromRawString(tm)
	if err != nil {
		t.Fatalf("Failed to add database task: %v", err)
	}

	// Verify the database task was created with appropriate tags
	tasks, err = tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	var dbTaskObj *Task
	for _, t := range tasks {
		if strings.Contains(t.Description, "database queries") {
			dbTaskObj = &t
			break
		}
	}

	if dbTaskObj == nil {
		t.Error("Database task not found in tasks")
	} else if dbTaskObj.Tags != "database" {
		t.Errorf("Expected database task to have 'database' tags, got '%s'", dbTaskObj.Tags)
	}

	// Test 4: Empty task should return error
	os.Args = []string{"test", "-task", "add", ""}
	_ = parseFlags()

	err = addTaskFromRawString(tm)
	if err == nil {
		t.Error("Expected error for empty task string")
	}

	// Test 5: Task with later keyword should get low priority
	lowPriorityTask := "Clean up old code files when we have time"
	os.Args = []string{"test", "-task", "add", lowPriorityTask}
	_ = parseFlags()

	err = addTaskFromRawString(tm)
	if err != nil {
		t.Fatalf("Failed to add low priority task: %v", err)
	}

	// Verify the low priority task
	tasks, err = tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	var lowPriorityTaskObj *Task
	for _, t := range tasks {
		if strings.Contains(t.Description, "Clean up old code") {
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

func TestAddTaskFromRawStringErrorHandling(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tasks.db")

	// Create task manager
	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet("", flag.ContinueOnError)

	// Test: No task arguments should return error
	os.Args = []string{"test", "-task", "add"}
	_ = parseFlags()

	err = addTaskFromRawString(tm)
	if err == nil {
		t.Error("Expected error for missing task string")
	}

	if !strings.Contains(err.Error(), "task add requires a task string") {
		t.Errorf("Expected error message about missing task string, got: %v", err)
	}
}

func TestTaskAddCommandIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tasks.db")

	// Create task manager
	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet("", flag.ContinueOnError)

	// Test integration with handleTaskCommand
	os.Args = []string{"test", "-task", "add", "Implement API endpoints for user management"}
	args := parseFlags()

	// This should successfully handle the add command
	err = handleTaskCommand(args.taskCommand)
	if err != nil {
		t.Fatalf("Failed to handle add command: %v", err)
	}

	// Verify task was created
	tasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task after integration test, got %d", len(tasks))
	}

	task := tasks[0]
	if !strings.Contains(task.Description, "API endpoints") {
		t.Errorf("Expected task description to contain 'API endpoints', got '%s'", task.Description)
	}

	// Verify metadata was set correctly
	if task.Metadata == "" {
		t.Error("Expected metadata to be set")
	}

	if !strings.Contains(task.Metadata, "task_add_command") {
		t.Errorf("Expected metadata to contain 'task_add_command', got '%s'", task.Metadata)
	}
}

func TestTaskAddPriorityHeuristics(t *testing.T) {
	testCases := []struct {
		description      string
		expectedPriority string
	}{
		{"Urgent fix needed ASAP", "high"},
		{"Critical security issue", "high"},
		{"Regular maintenance task", "medium"},
		{"Clean up code later", "low"},
		{"Do this when we have time", "low"},
		{"Standard feature implementation", "medium"},
		{"Emergency hotfix required", "high"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test_tasks.db")

			// Create task manager
			tm, err := NewTaskManager(dbPath)
			if err != nil {
				t.Fatalf("Failed to create task manager: %v", err)
			}
			defer tm.Close()

			// Reset flag state
			flag.CommandLine = flag.NewFlagSet("", flag.ContinueOnError)

			// Test the specific description
			os.Args = []string{"test", "-task", "add", tc.description}
			_ = parseFlags()

			err = addTaskFromRawString(tm)
			if err != nil {
				t.Fatalf("Failed to add task: %v", err)
			}

			// Verify priority
			tasks, err := tm.GetAllTasks()
			if err != nil {
				t.Fatalf("Failed to get all tasks: %v", err)
			}

			if len(tasks) != 1 {
				t.Fatalf("Expected 1 task, got %d", len(tasks))
			}

			task := tasks[0]
			if task.Priority != tc.expectedPriority {
				t.Errorf("Expected priority '%s' for description '%s', got '%s'",
					tc.expectedPriority, tc.description, task.Priority)
			}
		})
	}
}

func TestTaskAddTagHeuristics(t *testing.T) {
	testCases := []struct {
		description string
		expectedTag string
	}{
		{"Implement database connection", "database"},
		{"Create frontend UI components", "frontend"},
		{"Build backend API server", "backend"},
		{"Write unit tests for modules", "testing"},
		{"Regular task without specific keywords", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test_tasks.db")

			// Create task manager
			tm, err := NewTaskManager(dbPath)
			if err != nil {
				t.Fatalf("Failed to create task manager: %v", err)
			}
			defer tm.Close()

			// Reset flag state
			flag.CommandLine = flag.NewFlagSet("", flag.ContinueOnError)

			// Test the specific description
			os.Args = []string{"test", "-task", "add", tc.description}
			_ = parseFlags()

			err = addTaskFromRawString(tm)
			if err != nil {
				t.Fatalf("Failed to add task: %v", err)
			}

			// Verify tags
			tasks, err := tm.GetAllTasks()
			if err != nil {
				t.Fatalf("Failed to get all tasks: %v", err)
			}

			if len(tasks) != 1 {
				t.Fatalf("Expected 1 task, got %d", len(tasks))
			}

			task := tasks[0]
			if task.Tags != tc.expectedTag {
				t.Errorf("Expected tag '%s' for description '%s', got '%s'",
					tc.expectedTag, tc.description, task.Tags)
			}
		})
	}
}
