package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskVerification(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_verification.db")

	// Create task manager
	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test creating task ID 12 equivalent
	task, err := tm.CreateTaskWithStatus("test task to verify current behavior", "test task to verify current behavior", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create verification task: %v", err)
	}

	// Verify task was created with correct properties
	if task.Title != "test task to verify current behavior" {
		t.Errorf("Expected title 'test task to verify current behavior', got '%s'", task.Title)
	}

	if task.Description != "test task to verify current behavior" {
		t.Errorf("Expected description 'test task to verify current behavior', got '%s'", task.Description)
	}

	if task.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", task.Status)
	}

	if task.Priority != "medium" {
		t.Errorf("Expected priority 'medium', got '%s'", task.Priority)
	}

	// Test task retrieval
	retrievedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve task: %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Expected task ID %d, got %d", task.ID, retrievedTask.ID)
	}

	// Test task status updates
	err = tm.UpdateTaskStatus(task.ID, "in_progress")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	updatedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updatedTask.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", updatedTask.Status)
	}

	// Test task completion
	err = tm.UpdateTaskStatus(task.ID, "completed")
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	completedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get completed task: %v", err)
	}

	if completedTask.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", completedTask.Status)
	}

	if completedTask.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set when task is completed")
	}
}

func TestTaskBehaviorVerification(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_behavior.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Create test task to verify current behavior
	task, err := tm.CreateTaskWithStatus("test task to verify current behavior", "test task to verify current behavior", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Test 1: Verify task creation and retrieval work correctly
	allTasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(allTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(allTasks))
	}

	// Test 2: Verify completed tasks are excluded from GetAllTasks
	err = tm.UpdateTaskStatus(task.ID, "completed")
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	activeTasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get active tasks: %v", err)
	}

	// Should have zero tasks (completed task excluded)
	if len(activeTasks) != 0 {
		t.Errorf("Expected 0 active tasks, got %d", len(activeTasks))
	}

	// Test 3: Verify task history tracking
	historyTasks, err := tm.GetTasksByStatus("completed")
	if err != nil {
		t.Fatalf("Failed to get completed tasks: %v", err)
	}

	if len(historyTasks) != 1 {
		t.Errorf("Expected 1 completed task, got %d", len(historyTasks))
	}

	// Test 4: Verify task metadata updates
	metadata := `{"source": "task_add_command", "raw_input": "test task to verify current behavior", "analysis_date": "2026-01-19T14:10:25Z"}`
	err = tm.UpdateTaskMetadata(task.ID, "testing", metadata)
	if err != nil {
		t.Fatalf("Failed to update task metadata: %v", err)
	}

	updatedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task with updated metadata: %v", err)
	}

	if updatedTask.Tags != "testing" {
		t.Errorf("Expected tags 'testing', got '%s'", updatedTask.Tags)
	}

	if updatedTask.Metadata != metadata {
		t.Errorf("Expected metadata to be updated, got '%s'", updatedTask.Metadata)
	}
}

func TestTaskEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_edge_cases.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test 1: Empty title validation
	_, err = tm.CreateTask("", "description", "medium")
	if err == nil {
		t.Error("Expected error when creating task with empty title")
	}

	// Test 2: Invalid priority validation
	_, err = tm.CreateTask("title", "description", "invalid")
	if err == nil {
		t.Error("Expected error when creating task with invalid priority")
	}

	// Test 3: Getting non-existent task
	_, err = tm.GetTask(99999)
	if err == nil {
		t.Error("Expected error when getting non-existent task")
	}

	if !strings.Contains(err.Error(), "task not found") {
		t.Errorf("Expected 'task not found' error, got: %v", err)
	}

	// Test 4: Updating non-existent task
	err = tm.UpdateTaskStatus(99999, "completed")
	if err == nil {
		t.Error("Expected error when updating non-existent task")
	}

	// Test 5: Invalid status update
	task, err := tm.CreateTask("test task", "description", "medium")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	err = tm.UpdateTaskStatus(task.ID, "invalid_status")
	if err == nil {
		t.Error("Expected error when updating to invalid status")
	}

	// Test 6: Multiple field updates with validation
	updates := map[string]string{
		"title":       "",
		"priority":    "invalid",
		"description": "valid description",
	}

	err = tm.UpdateTaskMultiple(task.ID, updates)
	if err == nil {
		t.Error("Expected error when updating with invalid fields")
	}

	// Test 7: Task creation with timestamp verification
	beforeCreation := time.Now().Add(-time.Second)
	taskWithTime, err := tm.CreateTaskWithStatus("time test", "description", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	afterCreation := time.Now().Add(time.Second)

	if taskWithTime.CreatedAt.Before(beforeCreation) || taskWithTime.CreatedAt.After(afterCreation) {
		t.Errorf("CreatedAt time %v is outside expected range", taskWithTime.CreatedAt)
	}
}

func TestTaskConcurrencyAndConsistency(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_concurrency.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test 1: Create multiple tasks rapidly
	var taskIDs []int64
	for i := 0; i < 10; i++ {
		task, err := tm.CreateTaskWithStatus(
			"test task to verify current behavior",
			"description",
			"medium",
			"pending",
		)
		if err != nil {
			t.Fatalf("Failed to create task %d: %v", i, err)
		}
		taskIDs = append(taskIDs, task.ID)
	}

	// Verify all tasks were created with unique IDs
	uniqueIDs := make(map[int64]bool)
	for _, id := range taskIDs {
		if uniqueIDs[id] {
			t.Errorf("Duplicate task ID found: %d", id)
		}
		uniqueIDs[id] = true
	}

	// Test 2: Multiple status updates in sequence
	for i, id := range taskIDs {
		newStatus := "in_progress"
		if i%2 == 0 {
			newStatus = "completed"
		}

		err := tm.UpdateTaskStatus(id, newStatus)
		if err != nil {
			t.Fatalf("Failed to update task %d status: %v", id, err)
		}
	}

	// Verify final state
	stats, err := tm.GetTaskStats()
	if err != nil {
		t.Fatalf("Failed to get task stats: %v", err)
	}

	if stats["total"] != 10 {
		t.Errorf("Expected total task count 10, got %d", stats["total"])
	}

	// Test 3: Consistency after multiple operations
	tasks, err := tm.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	completedCount := 0
	inProgressCount := 0
	for _, task := range tasks {
		if task.Status == "completed" {
			completedCount++
		} else if task.Status == "in_progress" {
			inProgressCount++
		}
	}

	// Should only have in_progress tasks since completed are excluded
	if inProgressCount != 5 {
		t.Errorf("Expected 5 in_progress tasks, got %d", inProgressCount)
	}

	if completedCount != 0 {
		t.Errorf("Expected 0 completed tasks in GetAllTasks, got %d", completedCount)
	}
}

func TestTaskIntegrationBehavior(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_integration.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test creating task with tags and metadata (simulating researcher agent behavior)
	tags := "testing,verification,current_behavior"
	metadata := `{"source": "task_add_command", "raw_input": "test task to verify current behavior", "analysis_date": "2026-01-19T14:10:25Z"}`

	task, err := tm.CreateTaskWithStatus("test task to verify current behavior", "test task to verify current behavior", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Update with tags and metadata
	err = tm.UpdateTaskMetadata(task.ID, tags, metadata)
	if err != nil {
		t.Fatalf("Failed to update task metadata: %v", err)
	}

	// Verify task with complete workflow
	finalTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated task: %v", err)
	}

	// Test complete workflow: pending -> in_progress -> completed
	err = tm.UpdateTaskStatus(task.ID, "in_progress")
	if err != nil {
		t.Fatalf("Failed to set task to in_progress: %v", err)
	}

	// Simulate work being done
	time.Sleep(10 * time.Millisecond)

	err = tm.UpdateTaskStatus(task.ID, "completed")
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	// Final verification
	completedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get final task state: %v", err)
	}

	if completedTask.Status != "completed" {
		t.Errorf("Expected final status 'completed', got '%s'", completedTask.Status)
	}

	if completedTask.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}

	if finalTask.Tags != tags {
		t.Errorf("Expected tags '%s', got '%s'", tags, finalTask.Tags)
	}

	if finalTask.Metadata != metadata {
		t.Errorf("Expected metadata '%s', got '%s'", metadata, finalTask.Metadata)
	}
}
