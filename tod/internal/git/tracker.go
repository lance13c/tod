package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Tracker handles git repository operations for change detection
type Tracker struct {
	repoPath string
	isRepo   bool
}

// ChangeInfo represents information about a file change
type ChangeInfo struct {
	Path         string
	ChangeType   string // added, modified, deleted
	LastModified time.Time
	CommitHash   string
}

// NewTracker creates a new git tracker for the given repository path
func NewTracker(repoPath string) (*Tracker, error) {
	tracker := &Tracker{
		repoPath: repoPath,
	}

	// Check if this is a git repository
	gitDir := filepath.Join(repoPath, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		tracker.isRepo = true
	}

	return tracker, nil
}

// IsGitRepo returns true if the path is a git repository
func (t *Tracker) IsGitRepo() bool {
	return t.isRepo
}

// CurrentBranch returns the current git branch name
func (t *Tracker) CurrentBranch() string {
	if !t.isRepo {
		return ""
	}

	cmd := exec.Command("git", "-C", t.repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// LastCommit returns the hash of the last commit
func (t *Tracker) LastCommit() string {
	if !t.isRepo {
		return ""
	}

	cmd := exec.Command("git", "-C", t.repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// GetChangedFiles returns files that have changed since the last commit
func (t *Tracker) GetChangedFiles() ([]ChangeInfo, error) {
	if !t.isRepo {
		return nil, fmt.Errorf("not a git repository")
	}

	var changes []ChangeInfo

	// Get unstaged changes
	unstagedChanges, err := t.getUnstagedChanges()
	if err == nil {
		changes = append(changes, unstagedChanges...)
	}

	// Get staged changes
	stagedChanges, err := t.getStagedChanges()
	if err == nil {
		changes = append(changes, stagedChanges...)
	}

	return changes, nil
}

// getUnstagedChanges returns unstaged file changes
func (t *Tracker) getUnstagedChanges() ([]ChangeInfo, error) {
	cmd := exec.Command("git", "-C", t.repoPath, "diff", "--name-status")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return t.parseGitStatus(string(output))
}

// getStagedChanges returns staged file changes
func (t *Tracker) getStagedChanges() ([]ChangeInfo, error) {
	cmd := exec.Command("git", "-C", t.repoPath, "diff", "--cached", "--name-status")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return t.parseGitStatus(string(output))
}

// GetFileChanges returns changes for a specific file since last commit
func (t *Tracker) GetFileChanges(filePath string) (*ChangeInfo, error) {
	if !t.isRepo {
		return nil, fmt.Errorf("not a git repository")
	}

	// Check if file has been modified
	relPath, err := filepath.Rel(t.repoPath, filePath)
	if err != nil {
		relPath = filePath
	}

	cmd := exec.Command("git", "-C", t.repoPath, "diff", "--name-status", "HEAD", relPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(string(output)) == "" {
		// No changes detected
		return nil, nil
	}

	changes, err := t.parseGitStatus(string(output))
	if err != nil || len(changes) == 0 {
		return nil, err
	}

	return &changes[0], nil
}

// parseGitStatus parses git status output into ChangeInfo structs
func (t *Tracker) parseGitStatus(output string) ([]ChangeInfo, error) {
	var changes []ChangeInfo
	
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]

		var changeType string
		switch status {
		case "A":
			changeType = "added"
		case "M":
			changeType = "modified"
		case "D":
			changeType = "deleted"
		case "R":
			changeType = "renamed"
		case "C":
			changeType = "copied"
		default:
			changeType = "modified"
		}

		// Get file modification time
		fullPath := filepath.Join(t.repoPath, filePath)
		var modTime time.Time
		if info, err := os.Stat(fullPath); err == nil {
			modTime = info.ModTime()
		}

		changes = append(changes, ChangeInfo{
			Path:         filePath,
			ChangeType:   changeType,
			LastModified: modTime,
			CommitHash:   t.LastCommit(),
		})
	}

	return changes, scanner.Err()
}

// GetFilesChangedSince returns files changed since a specific commit
func (t *Tracker) GetFilesChangedSince(commitHash string) ([]ChangeInfo, error) {
	if !t.isRepo {
		return nil, fmt.Errorf("not a git repository")
	}

	cmd := exec.Command("git", "-C", t.repoPath, "diff", "--name-status", commitHash, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return t.parseGitStatus(string(output))
}

// IsFileTracked checks if a file is tracked by git
func (t *Tracker) IsFileTracked(filePath string) bool {
	if !t.isRepo {
		return false
	}

	relPath, err := filepath.Rel(t.repoPath, filePath)
	if err != nil {
		relPath = filePath
	}

	cmd := exec.Command("git", "-C", t.repoPath, "ls-files", relPath)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// GetIgnoredPatterns returns patterns from .gitignore
func (t *Tracker) GetIgnoredPatterns() ([]string, error) {
	gitignorePath := filepath.Join(t.repoPath, ".gitignore")
	
	file, err := os.Open(gitignorePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	return patterns, scanner.Err()
}

// IsFileIgnored checks if a file would be ignored by git
func (t *Tracker) IsFileIgnored(filePath string) bool {
	if !t.isRepo {
		return false
	}

	relPath, err := filepath.Rel(t.repoPath, filePath)
	if err != nil {
		relPath = filePath
	}

	cmd := exec.Command("git", "-C", t.repoPath, "check-ignore", relPath)
	err = cmd.Run()
	
	// If git check-ignore exits with status 0, the file is ignored
	return err == nil
}