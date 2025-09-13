package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lance13c/tod/internal/config"
	"github.com/lance13c/tod/internal/git"
	"github.com/lance13c/tod/internal/llm"
	"github.com/lance13c/tod/internal/types"
)

// ScanOptions configures the discovery process
type ScanOptions struct {
	Framework string // nextjs, express, gin, fastapi, auto
	Language  string // typescript, javascript, go, python, auto
	SkipLLM   bool   // Skip LLM analysis
	Silent    bool   // Suppress console output (for TUI mode)
}

// Scanner handles project discovery and analysis
type Scanner struct {
	projectRoot string
	options     ScanOptions
	config      *config.Config
	gitTracker  *git.Tracker
}

// ScanResults contains the results of a project scan
type ScanResults struct {
	Project    ProjectInfo         `json:"project"`
	Actions    []types.CodeAction  `json:"actions"`
	Files      []ScannedFile       `json:"files"`
	Git        GitInfo             `json:"git"`
	Errors     []ScanError         `json:"errors"`
	ScannedAt  time.Time           `json:"scanned_at"`
}

// ProjectInfo contains basic project metadata
type ProjectInfo struct {
	Name      string `json:"name"`
	Framework string `json:"framework"`
	Language  string `json:"language"`
	Root      string `json:"root"`
}


// ScannedFile represents a file that was analyzed
type ScannedFile struct {
	Path         string    `json:"path"`
	Language     string    `json:"language"`
	Framework    string    `json:"framework"`
	ActionCount  int       `json:"action_count"`
	LastModified time.Time `json:"last_modified"`
}

// GitInfo contains git repository information
type GitInfo struct {
	Tracking   bool   `json:"tracking"`
	Branch     string `json:"branch"`
	LastCommit string `json:"last_commit"`
}

// ScanError represents an error during scanning
type ScanError struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
	Level   string `json:"level"` // warning, error
}

// NewScanner creates a new project scanner
func NewScanner(projectRoot string, options ScanOptions, cfg *config.Config) *Scanner {
	gitTracker, _ := git.NewTracker(projectRoot)
	
	return &Scanner{
		projectRoot: projectRoot,
		options:     options,
		config:      cfg,
		gitTracker:  gitTracker,
	}
}

// ScanProject performs a universal project scan using LLM intelligence
func (s *Scanner) ScanProject() (*ScanResults, error) {
	if !s.options.Silent {
		fmt.Println("Analyzing codebase with AI...")
	}
	
	// Detect basic project info (language, name)
	project, err := s.detectProject()
	if err != nil {
		return nil, fmt.Errorf("failed to detect project: %w", err)
	}

	if !s.options.Silent {
		fmt.Printf("Detected: %s project\n", project.Language)
	}

	// Get git information
	gitInfo := s.getGitInfo()

	// Find ALL source files (framework-agnostic)
	files, err := s.findAllSourceFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find source files: %w", err)
	}

	fmt.Printf("Analyzing %d source files with AI...\n", len(files))

	// Estimate total cost for all files upfront
	if !s.options.SkipLLM {
		if shouldProceed, err := s.estimateTotalCostAndConfirm(files); err != nil {
			return nil, fmt.Errorf("failed to estimate costs: %w", err)
		} else if !shouldProceed {
			fmt.Printf("   ⏭️  Skipping LLM analysis for all files, using fallback\n")
			s.options.SkipLLM = true // Skip LLM for all remaining files
		}
	}

	// Analyze files for user actions using LLM
	var allActions []types.CodeAction
	var scanErrors []ScanError
	var scannedFiles []ScannedFile

	for i, file := range files {
		// Show progress counter that updates the same line
		if !s.options.Silent {
			fmt.Printf("\r   Analyzing files... [%d/%d] %s", i+1, len(files), filepath.Base(file))
		}
		
		actions, fileInfo, errs := s.analyzeFileWithLLM(file)
		allActions = append(allActions, actions...)
		scannedFiles = append(scannedFiles, fileInfo)
		scanErrors = append(scanErrors, errs...)
	}
	
	// Clear the progress line and show completion
	if !s.options.Silent {
		fmt.Printf("\r   Analyzing files... [%d/%d] Complete!%s\n", len(files), len(files), strings.Repeat(" ", 20))
		fmt.Printf("Discovered %d user actions\n", len(allActions))
	}

	// Create results
	results := &ScanResults{
		Project:   project,
		Actions:   allActions,
		Files:     scannedFiles,
		Git:       gitInfo,
		Errors:    scanErrors,
		ScannedAt: time.Now(),
	}

	return results, nil
}

// ScanProjectWithDirectories performs targeted scanning of specific directories
func (s *Scanner) ScanProjectWithDirectories(directories []string) (*ScanResults, error) {
	if !s.options.Silent {
		fmt.Println("Analyzing selected directories with AI...")
	}
	
	// Detect basic project info
	project, err := s.detectProject()
	if err != nil {
		return nil, fmt.Errorf("failed to detect project: %w", err)
	}

	if !s.options.Silent {
		fmt.Printf("Detected: %s project\n", project.Language)
	}

	// Get git information
	gitInfo := s.getGitInfo()

	// Find files in selected directories
	var files []string
	for _, dir := range directories {
		dirPath := filepath.Join(s.projectRoot, dir)
		dirFiles, err := s.findFilesInDirectory(dirPath)
		if err != nil {
			if !s.options.Silent {
				fmt.Printf("Could not scan directory %s: %v\n", dir, err)
			}
			continue
		}
		files = append(files, dirFiles...)
	}

	if len(files) == 0 {
		if !s.options.Silent {
			fmt.Println("No relevant files found in selected directories")
		}
		return &ScanResults{
			Project:   project,
			Actions:   []types.CodeAction{},
			Files:     []ScannedFile{},
			Git:       gitInfo,
			ScannedAt: time.Now(),
		}, nil
	}

	if !s.options.Silent {
		fmt.Printf("Analyzing %d source files in selected directories...\n", len(files))
	}

	// Analyze files for user actions using LLM
	var allActions []types.CodeAction
	var scanErrors []ScanError
	var scannedFiles []ScannedFile

	for i, file := range files {
		// Show progress counter that updates the same line
		if !s.options.Silent {
			fmt.Printf("\r   Analyzing files... [%d/%d] %s", i+1, len(files), filepath.Base(file))
		}
		
		actions, fileInfo, errs := s.analyzeFileWithLLM(file)
		allActions = append(allActions, actions...)
		scannedFiles = append(scannedFiles, fileInfo)
		scanErrors = append(scanErrors, errs...)
	}
	
	// Clear the progress line and show completion
	if !s.options.Silent {
		fmt.Printf("\r   Analyzing files... [%d/%d] Complete!%s\n", len(files), len(files), strings.Repeat(" ", 20))
		fmt.Printf("Discovered %d user actions\n", len(allActions))
	}

	// Create results
	results := &ScanResults{
		Project:   project,
		Actions:   allActions,
		Files:     scannedFiles,
		Git:       gitInfo,
		Errors:    scanErrors,
		ScannedAt: time.Now(),
	}

	return results, nil
}

// findFilesInDirectory finds relevant source files in a specific directory
func (s *Scanner) findFilesInDirectory(dirPath string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		if info.IsDir() {
			// Skip common directories that don't contain user-facing code
			if s.shouldSkipDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this is a source file we should analyze
		if s.isRelevantSourceFile(path) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// detectProject analyzes the project to determine basic info
func (s *Scanner) detectProject() (ProjectInfo, error) {
	project := ProjectInfo{
		Name: filepath.Base(s.projectRoot),
		Root: s.projectRoot,
	}

	// Only detect language - we don't need framework for discovery
	if s.options.Language == "auto" {
		project.Language = s.detectLanguage()
	} else {
		project.Language = s.options.Language
	}

	// Framework detection is no longer needed for discovery
	project.Framework = "detected-by-llm"

	return project, nil
}

// detectFramework tries to detect the web framework
func (s *Scanner) detectFramework() string {
	// Check for Next.js
	if s.fileExists("next.config.js") || s.fileExists("next.config.ts") {
		return "nextjs"
	}

	// Check for package.json dependencies
	if packageJSON := s.readPackageJSON(); packageJSON != nil {
		deps := packageJSON["dependencies"]
		devDeps := packageJSON["devDependencies"]
		
		if deps != nil || devDeps != nil {
			allDeps := make(map[string]interface{})
			if deps != nil {
				for k, v := range deps.(map[string]interface{}) {
					allDeps[k] = v
				}
			}
			if devDeps != nil {
				for k, v := range devDeps.(map[string]interface{}) {
					allDeps[k] = v
				}
			}

			if _, exists := allDeps["express"]; exists {
				return "express"
			}
			if _, exists := allDeps["fastify"]; exists {
				return "fastify"
			}
		}
	}

	// Check for Go
	if s.fileExists("go.mod") {
		content, err := os.ReadFile(filepath.Join(s.projectRoot, "go.mod"))
		if err == nil {
			if strings.Contains(string(content), "github.com/gin-gonic/gin") {
				return "gin"
			}
			if strings.Contains(string(content), "github.com/gorilla/mux") {
				return "gorilla"
			}
		}
		return "go-std"
	}

	// Check for Python
	if s.fileExists("requirements.txt") || s.fileExists("pyproject.toml") {
		if s.fileExists("main.py") {
			content, _ := os.ReadFile(filepath.Join(s.projectRoot, "main.py"))
			if strings.Contains(string(content), "from fastapi") {
				return "fastapi"
			}
			if strings.Contains(string(content), "from flask") {
				return "flask"
			}
		}
		return "python"
	}

	return "unknown"
}

// detectLanguage tries to detect the primary programming language
func (s *Scanner) detectLanguage() string {
	// Count file extensions
	extCounts := make(map[string]int)
	
	filepath.Walk(s.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		
		// Skip node_modules and other common ignore patterns
		if strings.Contains(path, "node_modules") || 
		   strings.Contains(path, ".git") ||
		   strings.Contains(path, "dist") ||
		   strings.Contains(path, "build") {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != "" {
			extCounts[ext]++
		}
		return nil
	})

	// Determine most common relevant extension
	if extCounts[".ts"] > extCounts[".js"] && extCounts[".ts"] > 0 {
		return "typescript"
	}
	if extCounts[".js"] > 0 {
		return "javascript"
	}
	if extCounts[".go"] > 0 {
		return "go"
	}
	if extCounts[".py"] > 0 {
		return "python"
	}

	return "unknown"
}

// Helper functions

func (s *Scanner) fileExists(filename string) bool {
	_, err := os.Stat(filepath.Join(s.projectRoot, filename))
	return err == nil
}

func (s *Scanner) readPackageJSON() map[string]interface{} {
	data, err := os.ReadFile(filepath.Join(s.projectRoot, "package.json"))
	if err != nil {
		return nil
	}

	var packageJSON map[string]interface{}
	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return nil
	}

	return packageJSON
}

func (s *Scanner) getGitInfo() GitInfo {
	if s.gitTracker == nil {
		return GitInfo{Tracking: false}
	}

	branch := s.gitTracker.CurrentBranch()
	commit := s.gitTracker.LastCommit()

	return GitInfo{
		Tracking:   true,
		Branch:     branch,
		LastCommit: commit,
	}
}

// findAllSourceFiles finds ALL relevant source files (framework-agnostic)
func (s *Scanner) findAllSourceFiles() ([]string, error) {
	var files []string
	
	// Walk the entire project directory
	err := filepath.Walk(s.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		if info.IsDir() {
			// Skip common directories that don't contain user-facing code
			if s.shouldSkipDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this is a source file we should analyze
		if s.isRelevantSourceFile(path) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (s *Scanner) shouldIncludeFile(filename string) bool {
	// Skip common ignore patterns
	ignorePatterns := []string{
		"node_modules",
		".git",
		"dist",
		"build",
		".next",
		"coverage",
		"__pycache__",
		".pytest_cache",
	}

	for _, pattern := range ignorePatterns {
		if strings.Contains(filename, pattern) {
			return false
		}
	}

	return true
}

// analyzeFileWithLLM uses LLM to discover actions in any source file
func (s *Scanner) analyzeFileWithLLM(filename string) ([]types.CodeAction, ScannedFile, []ScanError) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, ScannedFile{}, []ScanError{{
			File: filename,
			Message: err.Error(),
			Level: "error",
		}}
	}

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, ScannedFile{}, []ScanError{{
			File: filename,
			Message: fmt.Sprintf("Could not read file: %v", err),
			Level: "error",
		}}
	}

	// Extract codeContext from the file
	codeContextAnalyzer := NewContextAnalyzer()
	codeContext := codeContextAnalyzer.ExtractContext(string(content), filename)

	// Use LLM to discover user actions (framework-agnostic)
	actions := s.discoverActionsWithLLM(string(content), filename, codeContext)

	scannedFile := ScannedFile{
		Path:         filename,
		Language:     s.inferLanguageFromFile(filename),
		Framework:    "llm-analyzed", // No framework-specific analysis
		ActionCount:  len(actions),
		LastModified: fileInfo.ModTime(),
	}

	return actions, scannedFile, nil
}

// discoverActionsWithLLM sends code to LLM for universal action discovery
func (s *Scanner) discoverActionsWithLLM(content, filePath string, codeContext CodeContext) []types.CodeAction {
	// Skip LLM if disabled
	if s.options.SkipLLM {
		return s.fallbackActionDiscovery(content, filePath, codeContext)
	}

	// Try to create LLM client
	client, err := s.createLLMClient()
	if err != nil {
		// Only show detailed error once, not for every file
		// The error was already shown in estimateTotalCostAndConfirm
		return s.fallbackActionDiscovery(content, filePath, codeContext)
	}

	// Note: Cost estimation and confirmation now happens upfront in estimateTotalCostAndConfirm

	// Perform LLM analysis
	ctx := context.Background()
	analysis, err := client.AnalyzeCode(ctx, content, filePath)
	if err != nil {
		if !s.options.Silent {
			fmt.Printf("   LLM analysis failed (%v), using fallback\n", err)
		}
		return s.fallbackActionDiscovery(content, filePath, codeContext)
	}

	// Track usage
	if analysis.Usage != nil && !s.options.Silent {
		fmt.Printf("   Actual cost: %s (%s tokens)\n", 
			llm.FormatCost(analysis.Usage.TotalCost),
			llm.FormatTokens(analysis.Usage.TotalTokens))
		
		// Update usage tracking (import cmd package needed)
		// TODO: Move UpdateUsage to a shared package to avoid circular import
	}

	// Convert LLM analysis to TestActions
	return s.convertAnalysisToActions(analysis, filePath)
}

// fallbackActionDiscovery provides basic action discovery without LLM
func (s *Scanner) fallbackActionDiscovery(content, filePath string, codeContext CodeContext) []types.CodeAction {
	var actions []types.CodeAction

	// Use intelligent pattern recognition as fallback
	lowerContent := strings.ToLower(content)

	// Look for user-visible actions
	if s.containsAuthPatterns(content) {
		if strings.Contains(lowerContent, "magic") || strings.Contains(lowerContent, "passwordless") {
			actions = append(actions, types.CodeAction{
				ID:          "sign_in_magic_link",
				Name:        "Sign in with magic link",
				Category:    "Authentication",
				Type:        "form_submit",
				Description: "Request a passwordless sign-in link via email",
				Inputs: []types.UserInput{
					{Name: "email", Label: "Email address", Type: "email", Required: true, Example: "user@example.com"},
				},
				Expects: types.UserExpectation{
					Success:   "Magic link sent to email",
					Failure:   "Invalid email error shown",
					Validates: []string{"Email is valid", "Magic link sent"},
				},
				Implementation: types.TechnicalDetails{
					SourceFile: filePath,
				},
				LastModified: time.Now(),
			})
		}
	}

	// Add more pattern-based discovery...
	return actions
}

// Helper methods for the new approach

func (s *Scanner) shouldSkipDirectory(path string) bool {
	// Skip common directories that don't contain user-facing code
	skipDirs := []string{
		"node_modules", ".git", "dist", "build", ".next",
		"coverage", "__pycache__", ".pytest_cache", "vendor",
		".venv", "venv", "target", ".cargo",
	}

	dirName := filepath.Base(path)
	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}

	return false
}

func (s *Scanner) isRelevantSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Include common source file extensions
	relevantExts := []string{
		".js", ".jsx", ".ts", ".tsx",  // JavaScript/TypeScript
		".py",                         // Python
		".go",                         // Go
		".java", ".kt",               // JVM languages
		".rs",                        // Rust
		".php",                       // PHP
		".rb",                        // Ruby
		".cs",                        // C#
		".cpp", ".cc", ".c",          // C/C++
		".vue",                       // Vue.js
		".svelte",                    // Svelte
	}

	for _, relevantExt := range relevantExts {
		if ext == relevantExt {
			return true
		}
	}

	return false
}

func (s *Scanner) inferLanguageFromFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	languageMap := map[string]string{
		".js":     "javascript",
		".jsx":    "javascript", 
		".ts":     "typescript",
		".tsx":    "typescript",
		".py":     "python",
		".go":     "go",
		".java":   "java",
		".kt":     "kotlin",
		".rs":     "rust",
		".php":    "php",
		".rb":     "ruby",
		".cs":     "csharp",
		".cpp":    "cpp",
		".cc":     "cpp",
		".c":      "c",
		".vue":    "vue",
		".svelte": "svelte",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

func (s *Scanner) containsAuthPatterns(content string) bool {
	authKeywords := []string{
		"auth", "login", "signin", "signup", "register",
		"logout", "signout", "session", "password", "email",
		"magic", "token", "jwt", "oauth", "user", "account",
	}

	lowerContent := strings.ToLower(content)
	for _, keyword := range authKeywords {
		if strings.Contains(lowerContent, keyword) {
			return true
		}
	}

	return false
}

// createLLMClient creates an LLM client based on configuration
func (s *Scanner) createLLMClient() (llm.Client, error) {
	if s.config == nil {
		return nil, fmt.Errorf("no configuration available")
	}
	
	// Use configuration settings
	provider := llm.Provider(s.config.AI.Provider)
	apiKey := s.config.AI.APIKey
	
	if apiKey == "" {
		// Try to fallback to local analysis if API key is not configured
		if s.config.AI.Provider == "local" {
			return llm.NewClient(llm.Local, "", map[string]interface{}{})
		}
		return nil, fmt.Errorf("%s API key not configured - run 'tod init' to set up AI provider or use 'local' provider for free analysis", s.config.AI.Provider)
	}
	
	options := map[string]interface{}{
		"model": s.config.AI.Model,
	}
	
	return llm.NewClient(provider, apiKey, options)
}

// convertAnalysisToActions converts LLM analysis results to TestActions
func (s *Scanner) convertAnalysisToActions(analysis *llm.CodeAnalysis, filePath string) []types.CodeAction {
	var actions []types.CodeAction
	
	for _, endpoint := range analysis.Endpoints {
		action := types.CodeAction{
			ID:          generateActionID(endpoint.Path, endpoint.Method),
			Name:        endpoint.Description,
			Category:    "API",
			Type:        "api_request",
			Description: endpoint.Description,
			Implementation: types.TechnicalDetails{
				Endpoint:   endpoint.Path,
				Method:     endpoint.Method,
				SourceFile: fmt.Sprintf("%s:%d", filePath, endpoint.LineNumber),
			},
			LastModified: time.Now(),
		}
		
		actions = append(actions, action)
	}
	
	return actions
}

// estimateTotalCostAndConfirm estimates costs for all files and asks for confirmation once
func (s *Scanner) estimateTotalCostAndConfirm(files []string) (bool, error) {
	// Try to create LLM client
	client, err := s.createLLMClient()
	if err != nil {
		if !s.options.Silent {
			fmt.Printf("   🔄 LLM unavailable (%v)\n", err)
			fmt.Printf("   📁 Using free fallback analysis for %d files\n", len(files))
		}
		return false, nil // Not an error, just skip LLM
	}

	var totalTokens int64
	var totalCost float64

	// Estimate cost for each file
	for _, file := range files {
		// Read file to get size
		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}

		estimate := client.EstimateCost("analyze_code", len(content))
		totalTokens += estimate.TotalTokens
		totalCost += estimate.TotalCost
	}

	// Show total estimate
	if !s.options.Silent {
		fmt.Printf("   Total estimated cost: %s (%s tokens) for %d files\n", 
			llm.FormatCost(totalCost),
			llm.FormatTokens(totalTokens),
			len(files))

		// Ask for confirmation once
		fmt.Printf("   Continue with LLM analysis for all files? (y/N): ")
	}
	var response string
	if !s.options.Silent {
		fmt.Scanln(&response)
	} else {
		// In silent mode, skip LLM to avoid blocking on input
		return false, nil
	}
	
	return response == "y" || response == "Y", nil
}

