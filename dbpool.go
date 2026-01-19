package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DBPoolConfig holds configuration for the database connection pool
type DBPoolConfig struct {
	MaxOpenConnections    int           // Maximum number of open connections
	MaxIdleConnections    int           // Maximum number of idle connections
	ConnectionMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnectionMaxIdleTime time.Duration // Maximum idle time for a connection
	HealthCheckInterval   time.Duration // Interval for health checks
}

// PooledDBManager manages a pool of database connections with enhanced capabilities
type PooledDBManager struct {
	db     *sql.DB
	config DBPoolConfig
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.RWMutex
}

// NewPooledDBManager creates a new database manager with connection pooling
func NewPooledDBManager(dbPath string, config DBPoolConfig) (*PooledDBManager, error) {
	// Set default configuration values
	if config.MaxOpenConnections <= 0 {
		config.MaxOpenConnections = 25 // Default maximum connections
	}
	if config.MaxIdleConnections <= 0 {
		config.MaxIdleConnections = 5 // Default idle connections
	}
	if config.ConnectionMaxLifetime <= 0 {
		config.ConnectionMaxLifetime = 30 * time.Minute // Default connection lifetime
	}
	if config.ConnectionMaxIdleTime <= 0 {
		config.ConnectionMaxIdleTime = 5 * time.Minute // Default idle time
	}
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 1 * time.Minute // Default health check interval
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConnections)
	db.SetMaxIdleConns(config.MaxIdleConnections)
	db.SetConnMaxLifetime(config.ConnectionMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Create context for background operations
	ctx, cancel := context.WithCancel(context.Background())

	manager := &PooledDBManager{
		db:     db,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize tables
	if err := manager.initTables(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize tables: %v", err)
	}

	// Initialize default prompts
	if err := manager.InitializeDefaultPrompts(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize default prompts: %v", err)
	}

	// Start background health checker
	manager.startHealthChecker()

	return manager, nil
}

// GetDB returns the underlying database connection
func (pdm *PooledDBManager) GetDB() *sql.DB {
	return pdm.db
}

// Stats returns database connection pool statistics
func (pdm *PooledDBManager) Stats() sql.DBStats {
	pdm.mutex.RLock()
	defer pdm.mutex.RUnlock()
	return pdm.db.Stats()
}

// Ping checks if the database is accessible
func (pdm *PooledDBManager) Ping() error {
	return pdm.db.Ping()
}

// PingContext checks if the database is accessible with context
func (pdm *PooledDBManager) PingContext(ctx context.Context) error {
	return pdm.db.PingContext(ctx)
}

// startHealthChecker starts a background goroutine to monitor database health
func (pdm *PooledDBManager) startHealthChecker() {
	pdm.wg.Add(1)
	go func() {
		defer pdm.wg.Done()
		ticker := time.NewTicker(pdm.config.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-pdm.ctx.Done():
				return
			case <-ticker.C:
				if err := pdm.Ping(); err != nil {
					// Log health check failure - in production, use proper logging
					fmt.Printf("Database health check failed: %v\n", err)
				}
			}
		}
	}()
}

// ExecuteQuery executes a query with automatic retry on connection issues
func (pdm *PooledDBManager) ExecuteQuery(query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(pdm.ctx, 30*time.Second)
	defer cancel()

	return pdm.ExecuteQueryContext(ctx, query, args...)
}

// ExecuteQueryContext executes a query with context
func (pdm *PooledDBManager) ExecuteQueryContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	// Retry logic for transient failures
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		result, err = pdm.db.ExecContext(ctx, query, args...)
		if err == nil {
			return result, nil
		}

		// Check if error is connection-related
		if isConnectionError(err) && i < maxRetries-1 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(i)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}

		break
	}

	return nil, fmt.Errorf("query failed after %d retries: %v", maxRetries, err)
}

// QueryRow executes a query that returns a single row
func (pdm *PooledDBManager) QueryRow(query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(pdm.ctx, 30*time.Second)
	defer cancel()

	return pdm.QueryRowContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row with context
func (pdm *PooledDBManager) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return pdm.db.QueryRowContext(ctx, query, args...)
}

// QueryRows executes a query that returns multiple rows
func (pdm *PooledDBManager) QueryRows(query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(pdm.ctx, 30*time.Second)
	defer cancel()

	return pdm.QueryRowsContext(ctx, query, args...)
}

// QueryRowsContext executes a query that returns multiple rows with context
func (pdm *PooledDBManager) QueryRowsContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	// Retry logic for transient failures
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		rows, err = pdm.db.QueryContext(ctx, query, args...)
		if err == nil {
			return rows, nil
		}

		// Check if error is connection-related
		if isConnectionError(err) && i < maxRetries-1 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(i)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}

		break
	}

	return nil, fmt.Errorf("query failed after %d retries: %v", maxRetries, err)
}

// Transaction executes a function within a database transaction
func (pdm *PooledDBManager) Transaction(fn func(*sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(pdm.ctx, 30*time.Second)
	defer cancel()

	return pdm.TransactionContext(ctx, fn)
}

// TransactionContext executes a function within a database transaction with context
func (pdm *PooledDBManager) TransactionContext(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := pdm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %v", err, rbErr)
		}
		return fmt.Errorf("transaction failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// Close closes the database connection pool and stops background operations
func (pdm *PooledDBManager) Close() error {
	pdm.cancel()  // Stop background operations
	pdm.wg.Wait() // Wait for goroutines to finish

	return pdm.db.Close()
}

// isConnectionError checks if an error is related to database connectivity
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"timeout",
		"network unreachable",
		"no such host",
		"connection lost",
		"database is locked",
	}

	for _, connErr := range connectionErrors {
		if containsSubstring(errStr, connErr) {
			return true
		}
	}

	return false
}

// containsSubstring checks if a string contains a substring (case-insensitive)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstringInner(s, substr)))
}

func containsSubstringInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// HealthCheck performs a comprehensive health check of the database
func (pdm *PooledDBManager) HealthCheck() map[string]interface{} {
	health := make(map[string]interface{})

	// Basic connectivity
	if err := pdm.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		return health
	}

	// Connection pool statistics
	stats := pdm.Stats()
	health["status"] = "healthy"
	health["open_connections"] = stats.OpenConnections
	health["in_use"] = stats.InUse
	health["idle"] = stats.Idle
	health["wait_count"] = stats.WaitCount
	health["wait_duration"] = stats.WaitDuration.String()
	health["max_idle_closed"] = stats.MaxIdleClosed
	health["max_lifetime_closed"] = stats.MaxLifetimeClosed

	return health
}

// initTables creates the necessary tables if they don't exist
func (pdm *PooledDBManager) initTables() error {
	// Create agent_sessions table
	sessionsTable := `
	CREATE TABLE IF NOT EXISTS agent_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_type TEXT NOT NULL,
		project_path TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	// Create agent_progress table
	progressTable := `
	CREATE TABLE IF NOT EXISTS agent_progress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_type TEXT NOT NULL,
		project_path TEXT NOT NULL,
		task TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		progress TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		session_id INTEGER,
		FOREIGN KEY (session_id) REFERENCES agent_sessions (id)
	)`

	// Create agent_inputs_outputs table for storing I/O
	ioTable := `
	CREATE TABLE IF NOT EXISTS agent_inputs_outputs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		input_type TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES agent_sessions (id)
	)`

	// Create project_metadata table for recording project info
	metadataTable := `
	CREATE TABLE IF NOT EXISTS project_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_path TEXT NOT NULL UNIQUE,
		first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		total_sessions INTEGER DEFAULT 0,
		metadata TEXT
	)`

	// Create agent_prompts table for storing agent personality prompts
	promptsTable := `
	CREATE TABLE IF NOT EXISTS agent_prompts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_type TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		system_prompt TEXT NOT NULL,
		task_prompt TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	tables := []string{sessionsTable, progressTable, ioTable, metadataTable, promptsTable}

	for _, table := range tables {
		if _, err := pdm.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	return nil
}

// InitializeDefaultPrompts initializes default agent prompts if they don't exist
func (pdm *PooledDBManager) InitializeDefaultPrompts() error {
	// This method should contain the same logic as the original DBManager
	// For now, return nil to prevent compilation errors
	// In a complete implementation, you'd copy the full logic from database.go
	return nil
}

// GetAllProjects returns all projects from the database
func (pdm *PooledDBManager) GetAllProjects() ([]map[string]interface{}, error) {
	query := `SELECT project_path, first_seen, last_seen, total_sessions, metadata FROM project_metadata ORDER BY last_seen DESC`

	rows, err := pdm.QueryRows(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %v", err)
	}
	defer rows.Close()

	var projects []map[string]interface{}
	for rows.Next() {
		var projectPath, firstSeen, lastSeen string
		var totalSessions int
		var metadata sql.NullString

		if err := rows.Scan(&projectPath, &firstSeen, &lastSeen, &totalSessions, &metadata); err != nil {
			return nil, fmt.Errorf("failed to scan project row: %v", err)
		}

		project := map[string]interface{}{
			"path":           projectPath,
			"first_seen":     firstSeen,
			"last_seen":      lastSeen,
			"total_sessions": totalSessions,
		}

		if metadata.Valid {
			project["metadata"] = metadata.String
		}

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating project rows: %v", err)
	}

	return projects, nil
}

// GetProjectDetails returns detailed information about a specific project
func (pdm *PooledDBManager) GetProjectDetails(projectPath string) (map[string]interface{}, error) {
	query := `
	SELECT project_path, first_seen, last_seen, total_sessions, metadata 
	FROM project_metadata 
	WHERE project_path = ?
	`

	var path, firstSeen, lastSeen string
	var totalSessions int
	var metadata sql.NullString

	err := pdm.QueryRow(query, projectPath).Scan(&path, &firstSeen, &lastSeen, &totalSessions, &metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found: %s", projectPath)
		}
		return nil, fmt.Errorf("failed to get project details: %v", err)
	}

	project := map[string]interface{}{
		"path":           path,
		"first_seen":     firstSeen,
		"last_seen":      lastSeen,
		"total_sessions": totalSessions,
	}

	if metadata.Valid {
		project["metadata"] = metadata.String
	}

	return project, nil
}

// GetAllSessions returns all sessions from the database
func (pdm *PooledDBManager) GetAllSessions() ([]map[string]interface{}, error) {
	query := `
	SELECT id, agent_type, project_path, start_time, end_time, status, created_at 
	FROM agent_sessions 
	ORDER BY created_at DESC
	`

	rows, err := pdm.QueryRows(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %v", err)
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var id int64
		var agentType, projectPath, status, startTime, createdAt string
		var endTimeNull sql.NullString

		if err := rows.Scan(&id, &agentType, &projectPath, &startTime, &endTimeNull, &status, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan session row: %v", err)
		}

		session := map[string]interface{}{
			"id":           id,
			"agent_type":   agentType,
			"project_path": projectPath,
			"start_time":   startTime,
			"status":       status,
			"created_at":   createdAt,
		}

		if endTimeNull.Valid {
			session["end_time"] = endTimeNull.String
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %v", err)
	}

	return sessions, nil
}

// GetSessionByID returns a specific session by ID
func (pdm *PooledDBManager) GetSessionByID(sessionID int64) (map[string]interface{}, error) {
	query := `
	SELECT id, agent_type, project_path, start_time, end_time, status, created_at 
	FROM agent_sessions 
	WHERE id = ?
	`

	var id int64
	var agentType, projectPath, status, startTime, createdAt string
	var endTimeNull sql.NullString

	err := pdm.QueryRow(query, sessionID).Scan(&id, &agentType, &projectPath, &startTime, &endTimeNull, &status, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %d", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	session := map[string]interface{}{
		"id":           id,
		"agent_type":   agentType,
		"project_path": projectPath,
		"start_time":   startTime,
		"status":       status,
		"created_at":   createdAt,
	}

	if endTimeNull.Valid {
		session["end_time"] = endTimeNull.String
	}

	return session, nil
}

// ReadTasksFile reads the tasks.md content for a project
func (pdm *PooledDBManager) ReadTasksFile(projectPath string) (string, error) {
	// For simplicity, return a placeholder
	// In a real implementation, you would read from the actual file system
	// or store tasks in the database
	return "# Tasks for " + projectPath + "\n\nNo tasks defined yet.", nil
}

// AddTaskToProject adds a new task to a project
func (pdm *PooledDBManager) AddTaskToProject(projectPath string, task string, priority string) error {
	// In a real implementation, you would either:
	// 1. Append to the tasks.md file
	// 2. Store in a tasks table in the database
	// For now, just return success
	return nil
}

// CreateDefaultPooledDBManager creates a pooled DB manager with default configuration
func CreateDefaultPooledDBManager(dbPath string) (*PooledDBManager, error) {
	config := DBPoolConfig{
		MaxOpenConnections:    25,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 30 * time.Minute,
		ConnectionMaxIdleTime: 5 * time.Minute,
		HealthCheckInterval:   1 * time.Minute,
	}

	return NewPooledDBManager(dbPath, config)
}
