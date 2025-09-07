package database

import (
	"time"
)

// PageCapture represents a captured web page
type PageCapture struct {
	ID           int64     `db:"id"`
	URL          string    `db:"url"`
	Title        string    `db:"title"`
	HTMLFile     string    `db:"html_file"`
	HTMLLength   int       `db:"html_length"`
	CapturedAt   time.Time `db:"captured_at"`
	ChromePort   int       `db:"chrome_port"`
	WebSocketURL string    `db:"websocket_url"`
}

// DiscoveredAction represents an action found on a page
type DiscoveredAction struct {
	ID          int64     `db:"id"`
	CaptureID   int64     `db:"capture_id"`
	Description string    `db:"description"`
	Element     string    `db:"element"`
	Selector    string    `db:"selector"`
	Action      string    `db:"action"`
	IsTested    bool      `db:"is_tested"`
	Priority    string    `db:"priority"`
	CreatedAt   time.Time `db:"created_at"`
}

// TestGeneration represents a test generation session
type TestGeneration struct {
	ID         int64     `db:"id"`
	CaptureID  int64     `db:"capture_id"`
	Framework  string    `db:"framework"`
	TestCode   string    `db:"test_code"`
	FileName   string    `db:"file_name"`
	GeneratedAt time.Time `db:"generated_at"`
}

// LLMInteraction represents an interaction with the LLM
type LLMInteraction struct {
	ID           int64     `db:"id"`
	CaptureID    int64     `db:"capture_id"`
	InteractionType string `db:"interaction_type"` // "action_discovery", "test_generation", etc.
	Provider     string    `db:"provider"`
	Model        string    `db:"model"`
	Prompt       string    `db:"prompt"`
	Response     string    `db:"response"`
	TokensUsed   int       `db:"tokens_used"`
	Cost         float64   `db:"cost"`
	Error        string    `db:"error"`
	CreatedAt    time.Time `db:"created_at"`
}