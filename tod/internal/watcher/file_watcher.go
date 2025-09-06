package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ciciliostudio/tod/internal/git"
)

// FileWatcher monitors file system changes and triggers rescans
type FileWatcher struct {
	projectRoot string
	gitTracker  *git.Tracker
	watcher     *fsnotify.Watcher
	
	// Configuration
	debounceMS     int
	ignorePatterns []string
	
	// State
	mu           sync.RWMutex
	isWatching   bool
	pendingFiles map[string]time.Time
	
	// Callbacks
	onFileChanged func(files []string) error
}

// WatcherConfig configures the file watcher
type WatcherConfig struct {
	DebounceMS     int      `yaml:"debounce_ms"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

// DefaultConfig returns sensible defaults for file watching
func DefaultConfig() WatcherConfig {
	return WatcherConfig{
		DebounceMS: 500,
		IgnorePatterns: []string{
			"node_modules/**",
			".git/**",
			"dist/**", 
			"build/**",
			".next/**",
			"coverage/**",
			"__pycache__/**",
			".pytest_cache/**",
			"*.log",
			"*.tmp",
			".tif/cache/**",
			".tif/sessions/**",
		},
	}
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(projectRoot string, config WatcherConfig, gitTracker *git.Tracker) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	fw := &FileWatcher{
		projectRoot:    projectRoot,
		gitTracker:     gitTracker,
		watcher:        watcher,
		debounceMS:     config.DebounceMS,
		ignorePatterns: config.IgnorePatterns,
		pendingFiles:   make(map[string]time.Time),
	}

	return fw, nil
}

// SetChangeCallback sets the callback function for when files change
func (fw *FileWatcher) SetChangeCallback(callback func(files []string) error) {
	fw.onFileChanged = callback
}

// Start begins watching for file changes
func (fw *FileWatcher) Start(ctx context.Context) error {
	fw.mu.Lock()
	if fw.isWatching {
		fw.mu.Unlock()
		return fmt.Errorf("watcher is already running")
	}
	fw.isWatching = true
	fw.mu.Unlock()

	// Add project root to watcher
	if err := fw.addWatchPaths(); err != nil {
		return fmt.Errorf("failed to add watch paths: %w", err)
	}

	// Start debounce timer
	debounceTicker := time.NewTicker(time.Duration(fw.debounceMS) * time.Millisecond)
	defer debounceTicker.Stop()

	fmt.Printf("📁 Watching for file changes (debounce: %dms)\n", fw.debounceMS)

	for {
		select {
		case <-ctx.Done():
			fw.Stop()
			return ctx.Err()

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			if fw.shouldIgnoreEvent(event) {
				continue
			}

			fw.mu.Lock()
			fw.pendingFiles[event.Name] = time.Now()
			fw.mu.Unlock()

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed") 
			}
			fmt.Printf("⚠️  File watcher error: %v\n", err)

		case <-debounceTicker.C:
			// Process pending files after debounce period
			if err := fw.processPendingFiles(); err != nil {
				fmt.Printf("❌ Error processing file changes: %v\n", err)
			}
		}
	}
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.isWatching {
		fw.watcher.Close()
		fw.isWatching = false
		fmt.Println("📁 File watcher stopped")
	}
}

// IsWatching returns true if the watcher is currently active
func (fw *FileWatcher) IsWatching() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.isWatching
}

// addWatchPaths adds relevant directories to the watcher
func (fw *FileWatcher) addWatchPaths() error {
	// Add project root
	if err := fw.watcher.Add(fw.projectRoot); err != nil {
		return err
	}

	// Walk the directory tree and add relevant subdirectories
	return filepath.Walk(fw.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		if !info.IsDir() {
			return nil
		}

		// Skip ignored directories
		if fw.shouldIgnorePath(path) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := fw.watcher.Add(path); err != nil {
			// Log error but continue
			fmt.Printf("⚠️  Could not watch directory %s: %v\n", path, err)
		}

		return nil
	})
}

// shouldIgnoreEvent determines if a file system event should be ignored
func (fw *FileWatcher) shouldIgnoreEvent(event fsnotify.Event) bool {
	// Only watch for write and create events
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return true
	}

	return fw.shouldIgnorePath(event.Name)
}

// shouldIgnorePath checks if a path should be ignored based on patterns
func (fw *FileWatcher) shouldIgnorePath(path string) bool {
	// Make path relative to project root
	relPath, err := filepath.Rel(fw.projectRoot, path)
	if err != nil {
		relPath = path
	}

	// Check against ignore patterns
	for _, pattern := range fw.ignorePatterns {
		if fw.matchesPattern(relPath, pattern) {
			return true
		}
	}

	// Check if file is git-ignored
	if fw.gitTracker != nil && fw.gitTracker.IsFileIgnored(path) {
		return true
	}

	// Ignore hidden files and directories
	base := filepath.Base(relPath)
	if strings.HasPrefix(base, ".") && base != ".env" {
		return true
	}

	return false
}

// matchesPattern checks if a path matches a glob-style pattern
func (fw *FileWatcher) matchesPattern(path, pattern string) bool {
	// Handle ** patterns for recursive matching
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")
			
			if prefix != "" && !strings.HasPrefix(path, prefix) {
				return false
			}
			if suffix != "" && !strings.HasSuffix(path, suffix) {
				return false
			}
			return true
		}
	}

	// Use filepath.Match for simple patterns
	matched, _ := filepath.Match(pattern, path)
	return matched
}

// processPendingFiles handles debounced file changes
func (fw *FileWatcher) processPendingFiles() error {
	fw.mu.Lock()
	
	if len(fw.pendingFiles) == 0 {
		fw.mu.Unlock()
		return nil
	}

	// Get files that are old enough (past debounce period)
	debounceThreshold := time.Now().Add(-time.Duration(fw.debounceMS) * time.Millisecond)
	var filesToProcess []string
	
	for file, timestamp := range fw.pendingFiles {
		if timestamp.Before(debounceThreshold) {
			filesToProcess = append(filesToProcess, file)
			delete(fw.pendingFiles, file)
		}
	}
	
	fw.mu.Unlock()

	if len(filesToProcess) == 0 {
		return nil
	}

	// Filter to only relevant source files
	var relevantFiles []string
	for _, file := range filesToProcess {
		if fw.isRelevantSourceFile(file) {
			relevantFiles = append(relevantFiles, file)
		}
	}

	if len(relevantFiles) == 0 {
		return nil
	}

	fmt.Printf("📝 Detected changes in %d file(s)\n", len(relevantFiles))
	for _, file := range relevantFiles {
		relPath, _ := filepath.Rel(fw.projectRoot, file)
		fmt.Printf("   • %s\n", relPath)
	}

	// Call the change callback
	if fw.onFileChanged != nil {
		return fw.onFileChanged(relevantFiles)
	}

	return nil
}

// isRelevantSourceFile checks if a file is relevant for code analysis
func (fw *FileWatcher) isRelevantSourceFile(filePath string) bool {
	ext := filepath.Ext(filePath)
	
	// Common source file extensions
	sourceExts := []string{
		".js", ".jsx", ".ts", ".tsx",  // JavaScript/TypeScript
		".go",                         // Go
		".py",                         // Python
		".java", ".kt",               // JVM languages
		".rs",                        // Rust
		".php",                       // PHP
		".rb",                        // Ruby
		".c", ".cpp", ".h",           // C/C++
		".cs",                        // C#
		".yaml", ".yml", ".json",     // Configuration
	}

	for _, sourceExt := range sourceExts {
		if ext == sourceExt {
			return true
		}
	}

	return false
}

// GetWatchedPaths returns all currently watched paths
func (fw *FileWatcher) GetWatchedPaths() []string {
	return fw.watcher.WatchList()
}

// GetPendingFiles returns files waiting for debounce
func (fw *FileWatcher) GetPendingFiles() map[string]time.Time {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	
	result := make(map[string]time.Time)
	for k, v := range fw.pendingFiles {
		result[k] = v
	}
	return result
}