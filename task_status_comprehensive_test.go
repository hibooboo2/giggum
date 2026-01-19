package main

import (
	"path/filepath"
	"testing"
	"time"
)

// TestCompletedAtBugFix tests the specific CompletedAt timestamp issue
func TestCompletedAtBugFix(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_completed_at_bug.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Test 1: Create task and immediately mark as completed
	task, err := tm.CreateTaskWithStatus("test completed at bug", "description", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Record time before completion (truncated to seconds)
	beforeCompletion := time.Now().UTC().Truncate(time.Second)

	// Mark task as completed
	err = tm.UpdateTaskStatus(task.ID, "completed")
	if err != nil {
		t.Fatalf("Failed to mark task as completed: %v", err)
	}

	// Record time after completion (truncated to seconds with buffer)
	afterCompletion := time.Now().UTC().Truncate(time.Second).Add(2 * time.Second) // Add buffer for second precision

	// Retrieve the completed task
	completedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve completed task: %v", err)
	}

	// Verify CompletedAt is set
	if completedTask.CompletedAt == nil {
		t.Fatal("CompletedAt should be set when task is marked as completed")
	}

	// Verify CompletedAt is within expected time range
	if completedTask.CompletedAt.Before(beforeCompletion) || completedTask.CompletedAt.After(afterCompletion) {
		t.Errorf("CompletedAt time %v is outside expected range [%v, %v]",
			completedTask.CompletedAt, beforeCompletion, afterCompletion)
	}

	// Test 2: Change status from completed back to in_progress (should clear CompletedAt)
	err = tm.UpdateTaskStatus(task.ID, "in_progress")
	if err != nil {
		t.Fatalf("Failed to change task from completed to in_progress: %v", err)
	}

	reopenedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve reopened task: %v", err)
	}

	if reopenedTask.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", reopenedTask.Status)
	}

	// CompletedAt should be nil when task is no longer completed
	if reopenedTask.CompletedAt != nil {
		t.Errorf("CompletedAt should be nil when task is no longer completed, got %v", reopenedTask.CompletedAt)
	}

	// Test 3: Complete the task again (should set new CompletedAt)
	err = tm.UpdateTaskStatus(task.ID, "completed")
	if err != nil {
		t.Fatalf("Failed to complete task again: %v", err)
	}

	finalTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve final task: %v", err)
	}

	if finalTask.CompletedAt == nil {
		t.Fatal("CompletedAt should be set when task is completed again")
	}

	// Verify new CompletedAt is later than the previous one (if we had it)
	// Since it was cleared, just verify it's recent
	timeSinceCompletion := time.Since(*finalTask.CompletedAt)
	if timeSinceCompletion > 5*time.Second {
		t.Errorf("CompletedAt seems too old: %v ago", timeSinceCompletion)
	}
}

// TestTaskStatusTransitions tests all valid status transitions
func TestTaskStatusTransitions(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_status_transitions.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	validStatuses := []string{"pending", "in_progress", "completed", "cancelled"}

	// Test all possible status transitions
	for _, fromStatus := range validStatuses {
		for _, toStatus := range validStatuses {
			// Create task with initial status
			task, err := tm.CreateTaskWithStatus("transition test", "description", "medium", fromStatus)
			if err != nil {
				t.Fatalf("Failed to create task with status %s: %v", fromStatus, err)
			}

			// Update to new status
			err = tm.UpdateTaskStatus(task.ID, toStatus)
			if err != nil {
				t.Errorf("Failed to transition from %s to %s: %v", fromStatus, toStatus, err)
				continue
			}

			// Verify the transition worked
			updatedTask, err := tm.GetTask(task.ID)
			if err != nil {
				t.Errorf("Failed to retrieve task after transition from %s to %s: %v", fromStatus, toStatus, err)
				continue
			}

			if updatedTask.Status != toStatus {
				t.Errorf("Expected status %s after transition, got %s", toStatus, updatedTask.Status)
				continue
			}

			// Special validation for completed status
			if toStatus == "completed" {
				if updatedTask.CompletedAt == nil {
					t.Errorf("CompletedAt should be set when transitioning to completed (from %s)", fromStatus)
				}
			} else {
				if updatedTask.CompletedAt != nil {
					t.Errorf("CompletedAt should be nil when status is %s (transitioned from %s)", toStatus, fromStatus)
				}
			}

			// Clean up for next test
			tm.DeleteTask(task.ID)
		}
	}
}

// TestInvalidStatusUpdates tests error handling for invalid status values
func TestInvalidStatusUpdates(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_invalid_status.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	task, err := tm.CreateTask("test task", "description", "medium")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	invalidStatuses := []string{"", "invalid", "INVALID", "done", "ready", "blocked"}

	for _, invalidStatus := range invalidStatuses {
		err := tm.UpdateTaskStatus(task.ID, invalidStatus)
		if err == nil {
			t.Errorf("Expected error when updating to invalid status '%s', but got none", invalidStatus)
		}
	}
}

// TestTaskStatusHistory verifies that status changes are properly tracked in history
func TestTaskStatusHistory(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_status_history.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	task, err := tm.CreateTaskWithStatus("history test", "description", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Perform a series of status changes
	statusSequence := []string{"in_progress", "completed", "in_progress", "cancelled"}

	for i, newStatus := range statusSequence {
		err := tm.UpdateTaskStatus(task.ID, newStatus)
		if err != nil {
			t.Fatalf("Failed to update status to %s (step %d): %v", newStatus, i+1, err)
		}

		// Give a small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Verify final status
	finalTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get final task: %v", err)
	}

	if finalTask.Status != "cancelled" {
		t.Errorf("Expected final status 'cancelled', got '%s'", finalTask.Status)
	}

	if finalTask.CompletedAt != nil {
		t.Errorf("CompletedAt should be nil for cancelled task, got %v", finalTask.CompletedAt)
	}

	// TODO: Add verification of task_history table once we have a function to query it
	// For now, just ensure the status transitions worked without errors
}

// TestTaskStatusConcurrentUpdates tests handling of rapid status updates
func TestTaskStatusConcurrentUpdates(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_concurrent_status.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	task, err := tm.CreateTaskWithStatus("concurrent test", "description", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Perform rapid status updates
	statuses := []string{"in_progress", "completed", "in_progress", "pending"}

	for i, status := range statuses {
		err := tm.UpdateTaskStatus(task.ID, status)
		if err != nil {
			t.Errorf("Failed to update status to %s (iteration %d): %v", status, i, err)
		}
	}

	// Verify final state is consistent
	finalTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get final task state: %v", err)
	}

	if finalTask.Status != "pending" {
		t.Errorf("Expected final status 'pending', got '%s'", finalTask.Status)
	}

	if finalTask.Status == "pending" && finalTask.CompletedAt != nil {
		t.Errorf("CompletedAt should be nil for pending task, got %v", finalTask.CompletedAt)
	}
}
