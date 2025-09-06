package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ciciliostudio/tod/internal/discovery"
	"github.com/ciciliostudio/tod/internal/types"
)

// Manager handles manifest operations
type Manager struct {
	manifestPath string
}

// NewManager creates a new manifest manager
func NewManager(projectRoot string) *Manager {
	manifestPath := filepath.Join(projectRoot, ".tif", "manifest.json")
	return &Manager{
		manifestPath: manifestPath,
	}
}

// LoadManifest loads the manifest from disk
func (m *Manager) LoadManifest() (*discovery.ScanResults, error) {
	data, err := os.ReadFile(m.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest discovery.ScanResults
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// SaveManifest saves the manifest to disk
func (m *Manager) SaveManifest(results *discovery.ScanResults) error {
	return SaveManifest(m.manifestPath, results)
}

// SaveManifest saves scan results to a manifest file
func SaveManifest(manifestPath string, results *discovery.ScanResults) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	// Marshal to JSON with nice formatting
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write to file
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// UpdateAction updates a single action in the manifest
func (m *Manager) UpdateAction(actionID string, updatedAction types.CodeAction) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	// Find and update the action
	found := false
	for i, action := range manifest.Actions {
		if action.ID == actionID {
			manifest.Actions[i] = updatedAction
			found = true
			break
		}
	}

	if !found {
		// Add new action
		manifest.Actions = append(manifest.Actions, updatedAction)
	}

	// Update scan timestamp
	manifest.ScannedAt = time.Now()

	return m.SaveManifest(manifest)
}

// RemoveAction removes an action from the manifest
func (m *Manager) RemoveAction(actionID string) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	// Find and remove the action
	for i, action := range manifest.Actions {
		if action.ID == actionID {
			manifest.Actions = append(manifest.Actions[:i], manifest.Actions[i+1:]...)
			break
		}
	}

	// Update scan timestamp
	manifest.ScannedAt = time.Now()

	return m.SaveManifest(manifest)
}

// GetActionByID retrieves a specific action by ID
func (m *Manager) GetActionByID(actionID string) (*types.CodeAction, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, err
	}

	for _, action := range manifest.Actions {
		if action.ID == actionID {
			return &action, nil
		}
	}

	return nil, fmt.Errorf("action not found: %s", actionID)
}

// GetActionsByPath retrieves all actions for a specific path
func (m *Manager) GetActionsByPath(path string) ([]types.CodeAction, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, err
	}

	var actions []types.CodeAction
	for _, action := range manifest.Actions {
		if action.Implementation.Endpoint == path {
			actions = append(actions, action)
		}
	}

	return actions, nil
}

// GetStats returns statistics about the manifest
func (m *Manager) GetStats() (*ManifestStats, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, err
	}

	stats := &ManifestStats{
		TotalActions:    len(manifest.Actions),
		TotalFiles:      len(manifest.Files),
		Framework:       manifest.Project.Framework,
		Language:        manifest.Project.Language,
		LastScan:        manifest.ScannedAt,
		GitTracking:     manifest.Git.Tracking,
		ErrorCount:      len(manifest.Errors),
	}

	// Count actions by method
	stats.MethodCounts = make(map[string]int)
	for _, action := range manifest.Actions {
		stats.MethodCounts[action.Implementation.Method]++
	}

	// Count actions by path prefix
	stats.PathStats = make(map[string]int)
	for _, action := range manifest.Actions {
		prefix := getPathPrefix(action.Implementation.Endpoint)
		stats.PathStats[prefix]++
	}

	return stats, nil
}

// ManifestStats contains statistics about the manifest
type ManifestStats struct {
	TotalActions   int               `json:"total_actions"`
	TotalFiles     int               `json:"total_files"`
	Framework      string            `json:"framework"`
	Language       string            `json:"language"`
	LastScan       time.Time         `json:"last_scan"`
	GitTracking    bool              `json:"git_tracking"`
	ErrorCount     int               `json:"error_count"`
	MethodCounts   map[string]int    `json:"method_counts"`
	PathStats      map[string]int    `json:"path_stats"`
}

// IsStale checks if the manifest is potentially outdated
func (m *Manager) IsStale() (bool, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return true, err
	}

	// Consider stale if older than 1 hour
	staleThreshold := time.Hour
	return time.Since(manifest.ScannedAt) > staleThreshold, nil
}

// GetOutdatedFiles returns files that have been modified since last scan
func (m *Manager) GetOutdatedFiles() ([]string, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, err
	}

	var outdatedFiles []string
	for _, file := range manifest.Files {
		// Check if file has been modified since last scan
		if info, err := os.Stat(file.Path); err == nil {
			if info.ModTime().After(file.LastModified) {
				outdatedFiles = append(outdatedFiles, file.Path)
			}
		}
	}

	return outdatedFiles, nil
}

// Backup creates a backup of the current manifest
func (m *Manager) Backup() error {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := m.manifestPath + ".backup." + timestamp
	
	data, err := os.ReadFile(m.manifestPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// Helper functions

func getPathPrefix(path string) string {
	parts := filepath.SplitList(path)
	if len(parts) >= 2 {
		return "/" + parts[1]
	}
	return path
}