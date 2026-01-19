package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// DebugCompletedAtField tests the raw database values to understand the issue
func DebugCompletedAtField(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "debug_completed_at.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Create a task and mark as completed
	task, err := tm.CreateTaskWithStatus("debug test", "description", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	err = tm.UpdateTaskStatus(task.ID, "completed")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Query the raw database value
	query := `SELECT completed_at FROM tasks WHERE id = ?`
	var completedAtRaw sql.NullString
	err = tm.db.QueryRow(query, task.ID).Scan(&completedAtRaw)
	if err != nil {
		t.Fatalf("Failed to query raw completed_at: %v", err)
	}

	if completedAtRaw.Valid {
		t.Logf("Raw completed_at value: '%s'", completedAtRaw.String)

		// Try to parse with different formats
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			time.RFC3339,
			time.RFC3339Nano,
		}

		for _, format := range formats {
			if parsedTime, err := time.Parse(format, completedAtRaw.String); err == nil {
				t.Logf("Successfully parsed with format '%s': %v", format, parsedTime)
			} else {
				t.Logf("Failed to parse with format '%s': %v", format, err)
			}
		}
	} else {
		t.Log("completed_at is NULL in database")
	}

	// Now test what GetTask returns
	retrievedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrievedTask.CompletedAt != nil {
		t.Logf("GetTask returned CompletedAt: %v", *retrievedTask.CompletedAt)
	} else {
		t.Log("GetTask returned CompletedAt as nil")
	}
}

// TestCompletedAtFix tests if using time.Format() fixes the issue
func TestCompletedAtFix(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_completed_at_fix.db")

	tm, err := NewTaskManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create task manager: %v", err)
	}
	defer tm.Close()

	// Create a task
	task, err := tm.CreateTaskWithStatus("fix test", "description", "medium", "pending")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Manually update completed_at with formatted time
	formattedTime := time.Now().Format("2006-01-02 15:04:05")
	updateQuery := `UPDATE tasks SET status = 'completed', updated_at = CURRENT_TIMESTAMP, completed_at = ? WHERE id = ?`
	_, err = tm.db.Exec(updateQuery, formattedTime, task.ID)
	if err != nil {
		t.Fatalf("Failed to manually update task: %v", err)
	}

	// Test retrieval
	retrievedTask, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if retrievedTask.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", retrievedTask.Status)
	}

	if retrievedTask.CompletedAt == nil {
		t.Error("CompletedAt should be set with formatted time")
	} else {
		t.Logf("CompletedAt successfully set to: %v", *retrievedTask.CompletedAt)
	}
}
