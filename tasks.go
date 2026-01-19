package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FindGitRepositoryRoot finds the root directory of the git repository
// Returns empty string if not in a git repository or on error
func FindGitRepositoryRoot() string {
	// Use git rev-parse --show-toplevel to find git root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Trim whitespace and return
	gitRoot := strings.TrimSpace(string(output))
	if gitRoot == "" {
		return ""
	}

	// Verify it's actually a git repository by checking for .git directory
	gitDir := filepath.Join(gitRoot, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return ""
	}

	return gitRoot
}

// GetTaskDBPath determines the appropriate path for the tasks database
// If in a git repository, places it at the root, otherwise in current directory
func GetTaskDBPath() string {
	gitRoot := FindGitRepositoryRoot()
	if gitRoot != "" {
		return filepath.Join(gitRoot, "giggum_tasks.db")
	}
	return "./giggum_tasks.db"
}

// TaskManager manages tasks in SQLite database
type TaskManager struct {
	db *sql.DB
}

// Task represents a single task in the database
type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`   // pending, in_progress, completed, cancelled
	Priority    string     `json:"priority"` // high, medium, low
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Tags        string     `json:"tags,omitempty"`
	Metadata    string     `json:"metadata,omitempty"`
}

// NewTaskManager creates a new task manager with SQLite database
func NewTaskManager(dbPath string) (*TaskManager, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	manager := &TaskManager{db: db}

	// Initialize tables
	if err := manager.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %v", err)
	}

	return manager, nil
}

// initTables creates the necessary tables for task management
func (tm *TaskManager) initTables() error {
	// Create tasks table
	tasksTable := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		priority TEXT NOT NULL DEFAULT 'medium',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		tags TEXT,
		metadata TEXT
	)`

	// Create task_history table for tracking changes
	historyTable := `
	CREATE TABLE IF NOT EXISTS task_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL,
		action TEXT NOT NULL,
		old_status TEXT,
		new_status TEXT,
		notes TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
	)`

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id)",
	}

	tables := []string{tasksTable, historyTable}
	for _, table := range tables {
		if _, err := tm.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	for _, index := range indexes {
		if _, err := tm.db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %v", err)
		}
	}

	return nil
}

// Close closes the database connection
func (tm *TaskManager) Close() error {
	return tm.db.Close()
}

// CreateTask creates a new task in the database
func (tm *TaskManager) CreateTask(title, description, priority string) (*Task, error) {
	return tm.CreateTaskWithStatus(title, description, priority, "pending")
}

// CreateTaskWithStatus creates a new task in the database with specified status
func (tm *TaskManager) CreateTaskWithStatus(title, description, priority, status string) (*Task, error) {
	// Validate inputs
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}

	validPriorities := map[string]bool{"low": true, "medium": true, "high": true}
	if priority == "" {
		priority = "medium"
	} else if !validPriorities[strings.ToLower(priority)] {
		return nil, fmt.Errorf("invalid priority: %s (must be low, medium, or high)", priority)
	}

	validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "cancelled": true}
	if status == "" {
		status = "pending"
	} else if !validStatuses[strings.ToLower(status)] {
		return nil, fmt.Errorf("invalid status: %s (must be pending, in_progress, completed, or cancelled)", status)
	}

	query := `
	INSERT INTO tasks (title, description, priority, status)
	VALUES (?, ?, ?, ?)`

	result, err := tm.db.Exec(query, title, description, priority, status)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %v", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get task ID: %v", err)
	}

	// Return the created task
	return tm.GetTask(taskID)
}

// GetTask retrieves a task by ID
func (tm *TaskManager) GetTask(id int64) (*Task, error) {
	query := `
	SELECT id, title, description, status, priority, created_at, updated_at, completed_at, tags, metadata
	FROM tasks
	WHERE id = ?`

	row := tm.db.QueryRow(query, id)

	var task Task
	var completedAt sql.NullString
	var tags, metadata sql.NullString

	err := row.Scan(
		&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
		&task.CreatedAt, &task.UpdatedAt, &completedAt, &tags, &metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get task: %v", err)
	}

	// Handle nullable fields
	if completedAt.Valid {
		// Try multiple time formats
		formats := []string{
			"2006-01-02 15:04:05",
			time.RFC3339,
			time.RFC3339Nano,
		}
		for _, format := range formats {
			if t, err := time.Parse(format, completedAt.String); err == nil {
				task.CompletedAt = &t
				break
			}
		}
	}
	if tags.Valid {
		task.Tags = tags.String
	}
	if metadata.Valid {
		task.Metadata = metadata.String
	}

	return &task, nil
}

// GetAllTasks retrieves all tasks from the database
func (tm *TaskManager) GetAllTasks() ([]Task, error) {
	query := `
	SELECT id, title, description, status, priority, created_at, updated_at, completed_at, tags, metadata
	FROM tasks
	WHERE status != 'completed'
	ORDER BY priority DESC, created_at ASC`

	rows, err := tm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %v", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var completedAt sql.NullString
		var tags, metadata sql.NullString

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
			&task.CreatedAt, &task.UpdatedAt, &completedAt, &tags, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %v", err)
		}

		// Handle nullable fields
		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				task.CompletedAt = &t
			}
		}
		if tags.Valid {
			task.Tags = tags.String
		}
		if metadata.Valid {
			task.Metadata = metadata.String
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// UpdateTaskStatus updates the status of a task
func (tm *TaskManager) UpdateTaskStatus(id int64, newStatus string) error {
	// Validate status
	validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "cancelled": true}
	if !validStatuses[newStatus] {
		return fmt.Errorf("invalid status: %s. Must be one of: pending, in_progress, completed, cancelled", newStatus)
	}

	// Get current status first
	task, err := tm.GetTask(id)
	if err != nil {
		return fmt.Errorf("failed to get current task status: %v", err)
	}

	// Start transaction
	tx, err := tm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Update task status with proper completed_at handling
	updateQuery := `
	UPDATE tasks 
	SET status = ?, updated_at = CURRENT_TIMESTAMP`

	var args []interface{}
	args = append(args, newStatus)

	if newStatus == "completed" {
		updateQuery += ", completed_at = ?"
		args = append(args, time.Now().UTC().Format("2006-01-02 15:04:05"))
	} else {
		// Clear completed_at if status changed away from completed
		updateQuery += ", completed_at = NULL"
	}

	updateQuery += " WHERE id = ?"
	args = append(args, id)

	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to update task status: %v", err)
	}

	// Add to history
	historyQuery := `
	INSERT INTO task_history (task_id, action, old_status, new_status)
	VALUES (?, 'status_change', ?, ?)`

	_, err = tx.Exec(historyQuery, id, task.Status, newStatus)
	if err != nil {
		return fmt.Errorf("failed to add to history: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// DeleteTask deletes a task from the database
func (tm *TaskManager) DeleteTask(id int64) error {
	query := `DELETE FROM tasks WHERE id = ?`

	result, err := tm.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %d", id)
	}

	return nil
}

// GetTaskStats returns statistics about tasks
func (tm *TaskManager) GetTaskStats() (map[string]int, error) {
	stats := make(map[string]int)

	// Count by status
	statusQuery := `
	SELECT status, COUNT(*) 
	FROM tasks 
	GROUP BY status`

	rows, err := tm.db.Query(statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query task stats: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats[status] = count
	}

	// Get total count
	var total int
	err = tm.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&total)
	if err == nil {
		stats["total"] = total
	}

	return stats, nil
}

// UpdateTaskMetadata updates tags and metadata for a task
func (tm *TaskManager) UpdateTaskMetadata(id int64, tags, metadata string) error {
	query := `
	UPDATE tasks 
	SET tags = ?, metadata = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?`

	result, err := tm.db.Exec(query, tags, metadata, id)
	if err != nil {
		return fmt.Errorf("failed to update task metadata: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %d", id)
	}

	return nil
}

// GetTasksByStatus retrieves tasks filtered by status
func (tm *TaskManager) GetTasksByStatus(status string) ([]Task, error) {
	query := `
	SELECT id, title, description, status, priority, created_at, updated_at, completed_at, tags, metadata
	FROM tasks
	WHERE status = ?
	ORDER BY priority DESC, created_at ASC`

	rows, err := tm.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by status: %v", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var completedAt sql.NullString
		var tags, metadata sql.NullString

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
			&task.CreatedAt, &task.UpdatedAt, &completedAt, &tags, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %v", err)
		}

		// Handle nullable fields
		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				task.CompletedAt = &t
			}
		}
		if tags.Valid {
			task.Tags = tags.String
		}
		if metadata.Valid {
			task.Metadata = metadata.String
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// UpdateTaskPriority updates the priority of a task
func (tm *TaskManager) UpdateTaskPriority(id int64, newPriority string) error {
	// Validate priority
	validPriorities := map[string]bool{"high": true, "medium": true, "low": true}
	if !validPriorities[newPriority] {
		return fmt.Errorf("invalid priority: %s. Must be one of: high, medium, low", newPriority)
	}

	query := `
	UPDATE tasks 
	SET priority = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?`

	result, err := tm.db.Exec(query, newPriority, id)
	if err != nil {
		return fmt.Errorf("failed to update task priority: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %d", id)
	}

	return nil
}

// UpdateTaskTitle updates the title of a task
func (tm *TaskManager) UpdateTaskTitle(id int64, newTitle string) error {
	if strings.TrimSpace(newTitle) == "" {
		return fmt.Errorf("task title cannot be empty")
	}

	query := `
	UPDATE tasks 
	SET title = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?`

	result, err := tm.db.Exec(query, newTitle, id)
	if err != nil {
		return fmt.Errorf("failed to update task title: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %d", id)
	}

	return nil
}

// UpdateTaskDescription updates the description of a task
func (tm *TaskManager) UpdateTaskDescription(id int64, newDescription string) error {
	query := `
	UPDATE tasks 
	SET description = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?`

	result, err := tm.db.Exec(query, newDescription, id)
	if err != nil {
		return fmt.Errorf("failed to update task description: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %d", id)
	}

	return nil
}

// UpdateTaskMultiple updates multiple task properties in a single transaction
func (tm *TaskManager) UpdateTaskMultiple(id int64, updates map[string]string) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	// Get current task for validation and history
	task, err := tm.GetTask(id)
	if err != nil {
		return fmt.Errorf("failed to get current task: %v", err)
	}

	// Start transaction
	tx, err := tm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Build dynamic update query
	var setClauses []string
	var args []interface{}
	oldStatus := task.Status
	newStatus := task.Status
	statusChanged := false

	for field, value := range updates {
		switch field {
		case "status":
			// Validate status
			validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "cancelled": true}
			if !validStatuses[value] {
				return fmt.Errorf("invalid status: %s. Must be one of: pending, in_progress, completed, cancelled", value)
			}
			setClauses = append(setClauses, "status = ?")
			args = append(args, value)
			newStatus = value
			statusChanged = true
		case "priority":
			// Validate priority
			validPriorities := map[string]bool{"high": true, "medium": true, "low": true}
			if !validPriorities[value] {
				return fmt.Errorf("invalid priority: %s. Must be one of: high, medium, low", value)
			}
			setClauses = append(setClauses, "priority = ?")
			args = append(args, value)
		case "title":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("task title cannot be empty")
			}
			setClauses = append(setClauses, "title = ?")
			args = append(args, value)
		case "description":
			setClauses = append(setClauses, "description = ?")
			args = append(args, value)
		case "tags":
			setClauses = append(setClauses, "tags = ?")
			args = append(args, value)
		case "metadata":
			setClauses = append(setClauses, "metadata = ?")
			args = append(args, value)
		default:
			return fmt.Errorf("unknown field: %s", field)
		}
	}

	// Add updated_at timestamp
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	// Add completed_at if status changed to completed
	if statusChanged && newStatus == "completed" {
		setClauses = append(setClauses, "completed_at = ?")
		args = append(args, time.Now())
	} else if statusChanged && newStatus != "completed" {
		// Clear completed_at if status changed away from completed
		setClauses = append(setClauses, "completed_at = NULL")
	}

	// Add task ID to args
	args = append(args, id)

	// Build and execute update query
	updateQuery := fmt.Sprintf("UPDATE tasks SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to update task: %v", err)
	}

	// Add to history if status changed
	if statusChanged {
		historyQuery := `
		INSERT INTO task_history (task_id, action, old_status, new_status)
		VALUES (?, 'status_change', ?, ?)`

		_, err = tx.Exec(historyQuery, id, oldStatus, newStatus)
		if err != nil {
			return fmt.Errorf("failed to add to history: %v", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}
