package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Create database directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// InitSchema creates the database tables if they don't exist
func (db *DB) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS page_captures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		title TEXT NOT NULL,
		html_file TEXT NOT NULL,
		html_length INTEGER NOT NULL,
		captured_at TIMESTAMP NOT NULL,
		chrome_port INTEGER,
		websocket_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS discovered_actions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		capture_id INTEGER NOT NULL,
		description TEXT NOT NULL,
		element TEXT,
		selector TEXT,
		action TEXT NOT NULL,
		is_tested BOOLEAN DEFAULT 0,
		priority TEXT DEFAULT 'medium',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (capture_id) REFERENCES page_captures(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS test_generations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		capture_id INTEGER NOT NULL,
		framework TEXT NOT NULL,
		test_code TEXT,
		file_name TEXT,
		generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (capture_id) REFERENCES page_captures(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS llm_interactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		capture_id INTEGER,
		interaction_type TEXT NOT NULL,
		provider TEXT,
		model TEXT,
		prompt TEXT NOT NULL,
		response TEXT,
		tokens_used INTEGER,
		cost REAL,
		error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (capture_id) REFERENCES page_captures(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_captures_url ON page_captures(url);
	CREATE INDEX IF NOT EXISTS idx_captures_captured_at ON page_captures(captured_at);
	CREATE INDEX IF NOT EXISTS idx_actions_capture_id ON discovered_actions(capture_id);
	CREATE INDEX IF NOT EXISTS idx_actions_is_tested ON discovered_actions(is_tested);
	CREATE INDEX IF NOT EXISTS idx_generations_capture_id ON test_generations(capture_id);
	CREATE INDEX IF NOT EXISTS idx_llm_capture_id ON llm_interactions(capture_id);
	CREATE INDEX IF NOT EXISTS idx_llm_type ON llm_interactions(interaction_type);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// SavePageCapture saves a page capture to the database
func (db *DB) SavePageCapture(capture *PageCapture) (int64, error) {
	query := `
		INSERT INTO page_captures (url, title, html_file, html_length, captured_at, chrome_port, websocket_url)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, 
		capture.URL, 
		capture.Title, 
		capture.HTMLFile, 
		capture.HTMLLength,
		capture.CapturedAt,
		capture.ChromePort,
		capture.WebSocketURL,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save page capture: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return id, nil
}

// SaveDiscoveredActions saves multiple discovered actions
func (db *DB) SaveDiscoveredActions(captureID int64, actions []DiscoveredAction) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO discovered_actions (capture_id, description, element, selector, action, is_tested, priority)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, action := range actions {
		_, err := stmt.Exec(
			captureID,
			action.Description,
			action.Element,
			action.Selector,
			action.Action,
			action.IsTested,
			action.Priority,
		)
		if err != nil {
			return fmt.Errorf("failed to save action: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SaveTestGeneration saves a test generation record
func (db *DB) SaveTestGeneration(gen *TestGeneration) (int64, error) {
	query := `
		INSERT INTO test_generations (capture_id, framework, test_code, file_name)
		VALUES (?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query,
		gen.CaptureID,
		gen.Framework,
		gen.TestCode,
		gen.FileName,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save test generation: %w", err)
	}

	return result.LastInsertId()
}

// SaveLLMInteraction saves an LLM interaction to the database
func (db *DB) SaveLLMInteraction(interaction *LLMInteraction) (int64, error) {
	query := `
		INSERT INTO llm_interactions (
			capture_id, interaction_type, provider, model, 
			prompt, response, tokens_used, cost, error
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query,
		interaction.CaptureID,
		interaction.InteractionType,
		interaction.Provider,
		interaction.Model,
		interaction.Prompt,
		interaction.Response,
		interaction.TokensUsed,
		interaction.Cost,
		interaction.Error,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save LLM interaction: %w", err)
	}

	return result.LastInsertId()
}

// GetPageCapture retrieves a page capture by ID
func (db *DB) GetPageCapture(id int64) (*PageCapture, error) {
	query := `
		SELECT id, url, title, html_file, html_length, captured_at, chrome_port, websocket_url
		FROM page_captures
		WHERE id = ?
	`

	var capture PageCapture
	err := db.conn.QueryRow(query, id).Scan(
		&capture.ID,
		&capture.URL,
		&capture.Title,
		&capture.HTMLFile,
		&capture.HTMLLength,
		&capture.CapturedAt,
		&capture.ChromePort,
		&capture.WebSocketURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("capture not found")
		}
		return nil, fmt.Errorf("failed to get capture: %w", err)
	}

	return &capture, nil
}

// GetDiscoveredActions retrieves all actions for a capture
func (db *DB) GetDiscoveredActions(captureID int64) ([]DiscoveredAction, error) {
	query := `
		SELECT id, capture_id, description, element, selector, action, is_tested, priority, created_at
		FROM discovered_actions
		WHERE capture_id = ?
		ORDER BY priority DESC, id ASC
	`

	rows, err := db.conn.Query(query, captureID)
	if err != nil {
		return nil, fmt.Errorf("failed to query actions: %w", err)
	}
	defer rows.Close()

	var actions []DiscoveredAction
	for rows.Next() {
		var action DiscoveredAction
		err := rows.Scan(
			&action.ID,
			&action.CaptureID,
			&action.Description,
			&action.Element,
			&action.Selector,
			&action.Action,
			&action.IsTested,
			&action.Priority,
			&action.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan action: %w", err)
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// GetRecentCaptures retrieves the most recent page captures
func (db *DB) GetRecentCaptures(limit int) ([]PageCapture, error) {
	query := `
		SELECT id, url, title, html_file, html_length, captured_at, chrome_port, websocket_url
		FROM page_captures
		ORDER BY captured_at DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query captures: %w", err)
	}
	defer rows.Close()

	var captures []PageCapture
	for rows.Next() {
		var capture PageCapture
		err := rows.Scan(
			&capture.ID,
			&capture.URL,
			&capture.Title,
			&capture.HTMLFile,
			&capture.HTMLLength,
			&capture.CapturedAt,
			&capture.ChromePort,
			&capture.WebSocketURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan capture: %w", err)
		}
		captures = append(captures, capture)
	}

	return captures, nil
}

// GetUntestedActionCount returns the count of untested actions
func (db *DB) GetUntestedActionCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM discovered_actions WHERE is_tested = 0`
	err := db.conn.QueryRow(query).Scan(&count)
	return count, err
}

// GetStatistics returns database statistics
func (db *DB) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total captures
	var totalCaptures int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM page_captures").Scan(&totalCaptures)
	if err != nil {
		return nil, err
	}
	stats["total_captures"] = totalCaptures

	// Total actions
	var totalActions int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM discovered_actions").Scan(&totalActions)
	if err != nil {
		return nil, err
	}
	stats["total_actions"] = totalActions

	// Untested actions
	var untestedActions int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM discovered_actions WHERE is_tested = 0").Scan(&untestedActions)
	if err != nil {
		return nil, err
	}
	stats["untested_actions"] = untestedActions

	// Test generations
	var totalGenerations int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM test_generations").Scan(&totalGenerations)
	if err != nil {
		return nil, err
	}
	stats["total_generations"] = totalGenerations

	// LLM interactions
	var totalLLMInteractions int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM llm_interactions").Scan(&totalLLMInteractions)
	if err != nil {
		return nil, err
	}
	stats["llm_interactions"] = totalLLMInteractions

	// Most recent capture
	var lastCapture *time.Time
	err = db.conn.QueryRow("SELECT MAX(captured_at) FROM page_captures").Scan(&lastCapture)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if lastCapture != nil {
		stats["last_capture"] = lastCapture
	}

	return stats, nil
}

// GetLLMInteractions retrieves LLM interactions for a capture
func (db *DB) GetLLMInteractions(captureID int64) ([]LLMInteraction, error) {
	query := `
		SELECT id, capture_id, interaction_type, provider, model, 
		       prompt, response, tokens_used, cost, error, created_at
		FROM llm_interactions
		WHERE capture_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, captureID)
	if err != nil {
		return nil, fmt.Errorf("failed to query LLM interactions: %w", err)
	}
	defer rows.Close()

	var interactions []LLMInteraction
	for rows.Next() {
		var interaction LLMInteraction
		var errorStr sql.NullString
		err := rows.Scan(
			&interaction.ID,
			&interaction.CaptureID,
			&interaction.InteractionType,
			&interaction.Provider,
			&interaction.Model,
			&interaction.Prompt,
			&interaction.Response,
			&interaction.TokensUsed,
			&interaction.Cost,
			&errorStr,
			&interaction.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan LLM interaction: %w", err)
		}
		if errorStr.Valid {
			interaction.Error = errorStr.String
		}
		interactions = append(interactions, interaction)
	}

	return interactions, nil
}