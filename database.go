package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DBManager manages the SQLite database for agent sessions and progress
type DBManager struct {
	db *sql.DB
}

// NewDBManager creates a new database manager
func NewDBManager(dbPath string) (*DBManager, error) {
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

	manager := &DBManager{db: db}

	// Initialize tables
	if err := manager.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %v", err)
	}

	// Initialize default prompts
	if err := manager.InitializeDefaultPrompts(); err != nil {
		return nil, fmt.Errorf("failed to initialize default prompts: %v", err)
	}

	return manager, nil
}

// initTables creates the necessary tables if they don't exist
func (m *DBManager) initTables() error {
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
		if _, err := m.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	return nil
}

// Close closes the database connection
func (m *DBManager) Close() error {
	return m.db.Close()
}

// CreateAgentSession creates a new agent session
func (m *DBManager) CreateAgentSession(agentType AgentType, projectPath string) (int64, error) {
	query := `
	INSERT INTO agent_sessions (agent_type, project_path, start_time, status)
	VALUES (?, ?, ?, 'active')`

	result, err := m.db.Exec(query, string(agentType), projectPath, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to create agent session: %v", err)
	}

	sessionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get session ID: %v", err)
	}

	// Update project metadata
	if err := m.updateProjectMetadata(projectPath); err != nil {
		// Log error but don't fail the session creation
		fmt.Printf("Warning: failed to update project metadata: %v\n", err)
	}

	return sessionID, nil
}

// EndAgentSession marks an agent session as ended
func (m *DBManager) EndAgentSession(sessionID int64) error {
	query := `UPDATE agent_sessions SET end_time = ?, status = 'completed' WHERE id = ?`

	_, err := m.db.Exec(query, time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to end agent session: %v", err)
	}

	return nil
}

// AddSessionInput adds an input to a session
func (m *DBManager) AddSessionInput(sessionID int64, input string) error {
	query := `
	INSERT INTO agent_inputs_outputs (session_id, input_type, content)
	VALUES (?, 'input', ?)`

	_, err := m.db.Exec(query, sessionID, input)
	if err != nil {
		return fmt.Errorf("failed to add session input: %v", err)
	}

	return nil
}

// AddSessionOutput adds an output to a session
func (m *DBManager) AddSessionOutput(sessionID int64, output string) error {
	query := `
	INSERT INTO agent_inputs_outputs (session_id, input_type, content)
	VALUES (?, 'output', ?)`

	_, err := m.db.Exec(query, sessionID, output)
	if err != nil {
		return fmt.Errorf("failed to add session output: %v", err)
	}

	return nil
}

// CreateAgentProgress creates a new progress entry
func (m *DBManager) CreateAgentProgress(agentType AgentType, projectPath string, task string, sessionID int64) (int64, error) {
	query := `
	INSERT INTO agent_progress (agent_type, project_path, task, session_id)
	VALUES (?, ?, ?, ?)`

	result, err := m.db.Exec(query, string(agentType), projectPath, task, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to create agent progress: %v", err)
	}

	progressID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get progress ID: %v", err)
	}

	return progressID, nil
}

// UpdateAgentProgress updates an existing progress entry
func (m *DBManager) UpdateAgentProgress(progressID int64, status, progress string) error {
	query := `
	UPDATE agent_progress 
	SET status = ?, progress = ?, timestamp = ?
	WHERE id = ?`

	_, err := m.db.Exec(query, status, progress, time.Now(), progressID)
	if err != nil {
		return fmt.Errorf("failed to update agent progress: %v", err)
	}

	return nil
}

// GetAgentProgress retrieves progress for a specific agent type and project
func (m *DBManager) GetAgentProgress(agentType AgentType, projectPath string) ([]AgentProgress, error) {
	query := `
	SELECT id, agent_type, project_path, task, status, progress, timestamp
	FROM agent_progress
	WHERE agent_type = ? AND project_path = ?
	ORDER BY timestamp DESC`

	rows, err := m.db.Query(query, string(agentType), projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent progress: %v", err)
	}
	defer rows.Close()

	var progress []AgentProgress
	for rows.Next() {
		var p AgentProgress
		var timestampStr string

		err := rows.Scan(&p.ID, &p.AgentType, &p.ProjectPath, &p.Task, &p.Status, &p.Progress, &timestampStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan progress row: %v", err)
		}

		// Parse timestamp
		if timestampStr != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
				p.Timestamp = t
			}
		}

		progress = append(progress, p)
	}

	return progress, rows.Err()
}

// GetAgentSessions retrieves sessions for a specific agent type and project
func (m *DBManager) GetAgentSessions(agentType AgentType, projectPath string) ([]AgentSession, error) {
	query := `
	SELECT id, agent_type, project_path, start_time, end_time, status
	FROM agent_sessions
	WHERE agent_type = ? AND project_path = ?
	ORDER BY start_time DESC`

	rows, err := m.db.Query(query, string(agentType), projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent sessions: %v", err)
	}
	defer rows.Close()

	var sessions []AgentSession
	for rows.Next() {
		var s AgentSession
		var startTimeStr, endTimeStr sql.NullString

		err := rows.Scan(&s.ID, &s.AgentType, &s.ProjectPath, &startTimeStr, &endTimeStr, &s.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %v", err)
		}

		// Parse timestamps
		if startTimeStr.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", startTimeStr.String); err == nil {
				s.StartTime = t
			}
		}

		if endTimeStr.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", endTimeStr.String); err == nil {
				s.EndTime = &t
			}
		}

		// Get inputs and outputs for this session
		inputs, outputs, err := m.getSessionInputsOutputs(s.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get session inputs/outputs: %v", err)
		}

		s.Inputs = inputs
		s.Outputs = outputs

		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

// getSessionInputsOutputs retrieves inputs and outputs for a session
func (m *DBManager) getSessionInputsOutputs(sessionID int64) ([]string, []string, error) {
	// Get inputs
	inputQuery := `
	SELECT content FROM agent_inputs_outputs
	WHERE session_id = ? AND input_type = 'input'
	ORDER BY timestamp ASC`

	inputRows, err := m.db.Query(inputQuery, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query session inputs: %v", err)
	}
	defer inputRows.Close()

	var inputs []string
	for inputRows.Next() {
		var content string
		if err := inputRows.Scan(&content); err != nil {
			return nil, nil, fmt.Errorf("failed to scan input row: %v", err)
		}
		inputs = append(inputs, content)
	}

	// Get outputs
	outputQuery := `
	SELECT content FROM agent_inputs_outputs
	WHERE session_id = ? AND input_type = 'output'
	ORDER BY timestamp ASC`

	outputRows, err := m.db.Query(outputQuery, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query session outputs: %v", err)
	}
	defer outputRows.Close()

	var outputs []string
	for outputRows.Next() {
		var content string
		if err := outputRows.Scan(&content); err != nil {
			return nil, nil, fmt.Errorf("failed to scan output row: %v", err)
		}
		outputs = append(outputs, content)
	}

	return inputs, outputs, nil
}

// updateProjectMetadata updates or creates project metadata
func (m *DBManager) updateProjectMetadata(projectPath string) error {
	query := `
	INSERT OR REPLACE INTO project_metadata (project_path, last_seen, total_sessions)
	VALUES (?, 
		COALESCE((SELECT last_seen FROM project_metadata WHERE project_path = ?), CURRENT_TIMESTAMP),
		COALESCE((SELECT total_sessions FROM project_metadata WHERE project_path = ?), 0) + 1
	)`

	_, err := m.db.Exec(query, projectPath, projectPath, projectPath)
	if err != nil {
		return fmt.Errorf("failed to update project metadata: %v", err)
	}

	return nil
}

// GetGlobalDBPath returns the path to the global giggum database
func GetGlobalDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./giggum.db" // Fallback to local
	}
	return filepath.Join(homeDir, ".giggum", "giggum.db")
}

// GetProjectDBPath returns the path to the project-specific database
func GetProjectDBPath(projectPath string) string {
	return filepath.Join(projectPath, ".giggum.db")
}

// GetAllAgentProgress retrieves progress for all agent types in a project
func (m *DBManager) GetAllAgentProgress(projectPath string) (map[AgentType][]AgentProgress, error) {
	query := `
	SELECT id, agent_type, project_path, task, status, progress, timestamp
	FROM agent_progress
	WHERE project_path = ?
	ORDER BY agent_type, timestamp DESC`

	rows, err := m.db.Query(query, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query all agent progress: %v", err)
	}
	defer rows.Close()

	progress := make(map[AgentType][]AgentProgress)
	for rows.Next() {
		var p AgentProgress
		var timestampStr string

		err := rows.Scan(&p.ID, &p.AgentType, &p.ProjectPath, &p.Task, &p.Status, &p.Progress, &timestampStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan progress row: %v", err)
		}

		// Parse timestamp
		if timestampStr != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
				p.Timestamp = t
			}
		}

		progress[p.AgentType] = append(progress[p.AgentType], p)
	}

	return progress, rows.Err()
}

// GetProjectStats retrieves statistics for a project
func (m *DBManager) GetProjectStats(projectPath string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total sessions per agent type
	sessionQuery := `
	SELECT agent_type, COUNT(*) as session_count
	FROM agent_sessions
	WHERE project_path = ?
	GROUP BY agent_type`

	sessionRows, err := m.db.Query(sessionQuery, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query session stats: %v", err)
	}
	defer sessionRows.Close()

	sessionStats := make(map[string]int)
	for sessionRows.Next() {
		var agentType string
		var count int
		if err := sessionRows.Scan(&agentType, &count); err != nil {
			continue
		}
		sessionStats[agentType] = count
	}
	stats["sessions"] = sessionStats

	// Get progress stats per agent type
	progressQuery := `
	SELECT agent_type, status, COUNT(*) as count
	FROM agent_progress
	WHERE project_path = ?
	GROUP BY agent_type, status`

	progressRows, err := m.db.Query(progressQuery, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query progress stats: %v", err)
	}
	defer progressRows.Close()

	progressStats := make(map[string]map[string]int)
	for progressRows.Next() {
		var agentType, status string
		var count int
		if err := progressRows.Scan(&agentType, &status, &count); err != nil {
			continue
		}
		if progressStats[agentType] == nil {
			progressStats[agentType] = make(map[string]int)
		}
		progressStats[agentType][status] = count
	}
	stats["progress"] = progressStats

	return stats, nil
}

// StoreAgentPrompt stores or updates an agent prompt in the database
func (m *DBManager) StoreAgentPrompt(prompt AgentPrompt) error {
	query := `
	INSERT OR REPLACE INTO agent_prompts 
	(agent_type, name, description, system_prompt, task_prompt, updated_at)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := m.db.Exec(query, string(prompt.Type), prompt.Name, prompt.Description, prompt.SystemPrompt, prompt.TaskPrompt)
	if err != nil {
		return fmt.Errorf("failed to store agent prompt: %v", err)
	}

	return nil
}

// GetStoredAgentPrompt retrieves an agent prompt from the database
func (m *DBManager) GetStoredAgentPrompt(agentType AgentType) (AgentPrompt, error) {
	query := `
	SELECT agent_type, name, description, system_prompt, task_prompt
	FROM agent_prompts
	WHERE agent_type = ?`

	row := m.db.QueryRow(query, string(agentType))

	var prompt AgentPrompt
	err := row.Scan(&prompt.Type, &prompt.Name, &prompt.Description, &prompt.SystemPrompt, &prompt.TaskPrompt)
	if err != nil {
		if err == sql.ErrNoRows {
			return AgentPrompt{}, fmt.Errorf("agent prompt for type %s not found", agentType)
		}
		return AgentPrompt{}, fmt.Errorf("failed to retrieve agent prompt: %v", err)
	}

	return prompt, nil
}

// GetAllStoredAgentPrompts retrieves all agent prompts from the database
func (m *DBManager) GetAllStoredAgentPrompts() (map[AgentType]AgentPrompt, error) {
	query := `
	SELECT agent_type, name, description, system_prompt, task_prompt
	FROM agent_prompts`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent prompts: %v", err)
	}
	defer rows.Close()

	prompts := make(map[AgentType]AgentPrompt)
	for rows.Next() {
		var prompt AgentPrompt
		err := rows.Scan(&prompt.Type, &prompt.Name, &prompt.Description, &prompt.SystemPrompt, &prompt.TaskPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan prompt row: %v", err)
		}
		prompts[prompt.Type] = prompt
	}

	return prompts, rows.Err()
}

// InitializeDefaultPrompts stores the default agent prompts in the database
func (m *DBManager) InitializeDefaultPrompts() error {
	defaultPrompts := GetAgentPrompts()

	for agentType, prompt := range defaultPrompts {
		// Check if prompt already exists
		_, err := m.GetStoredAgentPrompt(agentType)
		if err != nil {
			// Prompt doesn't exist, store it
			if err := m.StoreAgentPrompt(prompt); err != nil {
				return fmt.Errorf("failed to initialize prompt for %s: %v", agentType, err)
			}
		}
	}

	return nil
}
