package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ciciliostudio/tod/internal/ui/components"
)

// E2EFramework represents a detected or configured E2E testing framework
type E2EFramework struct {
	Name         string            `json:"name"`          // "playwright", "cypress", "selenium"
	DisplayName  string            `json:"display_name"`  // "Playwright"
	Version      string            `json:"version"`       // "1.40.0"
	Language     string            `json:"language"`      // "typescript", "javascript", "python"
	RunCommand   string            `json:"run_command"`   // "npx playwright test"
	ConfigFile   string            `json:"config_file"`   // "playwright.config.ts"
	TestDir      string            `json:"test_dir"`      // "tests/e2e"
	Extensions   []string          `json:"extensions"`    // [".spec.ts", ".test.ts"]
	Custom       bool              `json:"custom"`        // true if user-defined
	CustomConfig map[string]string `json:"custom_config,omitempty"` // Custom framework config
}

// FrameworkDetector handles automatic detection of E2E testing frameworks
type FrameworkDetector struct {
	projectRoot string
}

// NewFrameworkDetector creates a new framework detector
func NewFrameworkDetector(projectRoot string) *FrameworkDetector {
	return &FrameworkDetector{
		projectRoot: projectRoot,
	}
}

// DetectFramework attempts to automatically detect the E2E framework
func (fd *FrameworkDetector) DetectFramework() (*E2EFramework, error) {
	// Try different detection methods in order of reliability
	
	// 1. Check package.json for framework dependencies
	if framework := fd.checkPackageJSON(); framework != nil {
		fd.enhanceWithConfigFiles(framework)
		return framework, nil
	}
	
	// 2. Check for framework config files
	if framework := fd.checkConfigFiles(); framework != nil {
		return framework, nil
	}
	
	// 3. Check test directories for framework patterns
	if framework := fd.checkTestDirectories(); framework != nil {
		return framework, nil
	}
	
	// Framework not detected
	return nil, fmt.Errorf("could not auto-detect E2E testing framework")
}

// checkPackageJSON examines package.json for framework dependencies
func (fd *FrameworkDetector) checkPackageJSON() *E2EFramework {
	packageJSONPath := filepath.Join(fd.projectRoot, "package.json")
	
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil // No package.json found
	}
	
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
	}
	
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	
	// Merge dependencies for checking
	allDeps := make(map[string]string)
	for name, version := range pkg.Dependencies {
		allDeps[name] = version
	}
	for name, version := range pkg.DevDependencies {
		allDeps[name] = version
	}
	
	// Framework detection patterns
	detectionMap := map[string]*E2EFramework{
		"@playwright/test": {
			Name:        "playwright",
			DisplayName: "Playwright",
			RunCommand:  "npx playwright test",
			ConfigFile:  "playwright.config.ts",
			TestDir:     "tests",
			Extensions:  []string{".spec.ts", ".spec.js", ".test.ts", ".test.js"},
		},
		"cypress": {
			Name:        "cypress",
			DisplayName: "Cypress", 
			RunCommand:  "npx cypress run",
			ConfigFile:  "cypress.config.js",
			TestDir:     "cypress/e2e",
			Extensions:  []string{".cy.ts", ".cy.js", ".spec.ts", ".spec.js"},
		},
		"selenium-webdriver": {
			Name:        "selenium",
			DisplayName: "Selenium WebDriver",
			RunCommand:  "npm test",
			TestDir:     "test",
			Extensions:  []string{".test.js", ".spec.js"},
		},
		"puppeteer": {
			Name:        "puppeteer",
			DisplayName: "Puppeteer",
			RunCommand:  "npm test",
			TestDir:     "test",
			Extensions:  []string{".test.js", ".spec.js"},
		},
		"webdriverio": {
			Name:        "webdriverio",
			DisplayName: "WebdriverIO",
			RunCommand:  "npx wdio run",
			ConfigFile:  "wdio.conf.js",
			TestDir:     "test/specs",
			Extensions:  []string{".e2e.js", ".spec.js"},
		},
		"nightwatch": {
			Name:        "nightwatch",
			DisplayName: "Nightwatch.js",
			RunCommand:  "npx nightwatch",
			ConfigFile:  "nightwatch.conf.js",
			TestDir:     "tests",
			Extensions:  []string{".js"},
		},
		"testcafe": {
			Name:        "testcafe",
			DisplayName: "TestCafe",
			RunCommand:  "npx testcafe",
			TestDir:     "tests",
			Extensions:  []string{".js", ".ts"},
		},
	}
	
	// Check for framework dependencies
	for dep, framework := range detectionMap {
		if version, exists := allDeps[dep]; exists {
			framework.Version = cleanVersion(version)
			
			// Detect language from project structure
			framework.Language = fd.detectLanguage()
			
			// Try to infer run command from scripts
			if runCmd := fd.inferRunCommand(pkg.Scripts, framework.Name); runCmd != "" {
				framework.RunCommand = runCmd
			}
			
			return framework
		}
	}
	
	return nil
}

// checkConfigFiles looks for framework-specific configuration files
func (fd *FrameworkDetector) checkConfigFiles() *E2EFramework {
	configPatterns := map[string]*E2EFramework{
		"playwright.config.*": {
			Name:        "playwright",
			DisplayName: "Playwright",
			RunCommand:  "npx playwright test",
		},
		"cypress.config.*": {
			Name:        "cypress", 
			DisplayName: "Cypress",
			RunCommand:  "npx cypress run",
		},
		"wdio.conf.*": {
			Name:        "webdriverio",
			DisplayName: "WebdriverIO", 
			RunCommand:  "npx wdio run",
		},
		"nightwatch.conf.*": {
			Name:        "nightwatch",
			DisplayName: "Nightwatch.js",
			RunCommand:  "npx nightwatch",
		},
		"testcafe.config.*": {
			Name:        "testcafe",
			DisplayName: "TestCafe",
			RunCommand:  "npx testcafe",
		},
		"jest-puppeteer.config.*": {
			Name:        "jest-puppeteer",
			DisplayName: "Jest + Puppeteer",
			RunCommand:  "npm test",
		},
	}
	
	for pattern, framework := range configPatterns {
		if files := fd.globFiles(pattern); len(files) > 0 {
			framework.ConfigFile = files[0]
			framework.Language = fd.detectLanguage()
			return framework
		}
	}
	
	return nil
}

// checkTestDirectories examines test directory structures for framework patterns
func (fd *FrameworkDetector) checkTestDirectories() *E2EFramework {
	testDirs := []string{
		"tests", "test", "e2e", "cypress", "__tests__", "spec",
	}
	
	for _, dir := range testDirs {
		testPath := filepath.Join(fd.projectRoot, dir)
		if _, err := os.Stat(testPath); err != nil {
			continue
		}
		
		// Analyze files in test directory
		if framework := fd.analyzeTestFiles(testPath); framework != nil {
			framework.TestDir = dir
			framework.Language = fd.detectLanguage()
			return framework
		}
	}
	
	return nil
}

// analyzeTestFiles examines test files to infer the framework
func (fd *FrameworkDetector) analyzeTestFiles(testDir string) *E2EFramework {
	var allFiles []string
	
	err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		if !info.IsDir() && fd.isTestFile(path) {
			allFiles = append(allFiles, path)
		}
		
		return nil
	})
	
	if err != nil || len(allFiles) == 0 {
		return nil
	}
	
	// Analyze first few test files for framework patterns
	maxFiles := 5
	if len(allFiles) < maxFiles {
		maxFiles = len(allFiles)
	}
	
	frameworkVotes := make(map[string]int)
	
	for i := 0; i < maxFiles; i++ {
		content, err := os.ReadFile(allFiles[i])
		if err != nil {
			continue
		}
		
		framework := fd.inferFrameworkFromCode(string(content))
		if framework != "" {
			frameworkVotes[framework]++
		}
	}
	
	// Return the framework with most votes
	var bestFramework string
	var maxVotes int
	
	for framework, votes := range frameworkVotes {
		if votes > maxVotes {
			maxVotes = votes
			bestFramework = framework
		}
	}
	
	if bestFramework != "" {
		return fd.createFrameworkFromName(bestFramework)
	}
	
	return nil
}

// Helper methods

func (fd *FrameworkDetector) globFiles(pattern string) []string {
	var files []string
	
	// Simple glob implementation for config files
	basePattern := strings.ReplaceAll(pattern, ".*", "")
	extensions := []string{".js", ".ts", ".json", ".config.js", ".config.ts"}
	
	for _, ext := range extensions {
		filePath := filepath.Join(fd.projectRoot, basePattern+ext)
		if _, err := os.Stat(filePath); err == nil {
			files = append(files, basePattern+ext)
		}
	}
	
	return files
}

func (fd *FrameworkDetector) isTestFile(filePath string) bool {
	testPatterns := []string{
		".spec.", ".test.", ".e2e.", ".cy.", "_test.", "_spec.",
	}
	
	lowerPath := strings.ToLower(filePath)
	for _, pattern := range testPatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}
	
	return false
}

func (fd *FrameworkDetector) inferFrameworkFromCode(content string) string {
	lowerContent := strings.ToLower(content)
	
	// Framework-specific patterns
	patterns := map[string][]string{
		"playwright": {
			"@playwright/test", "test(", "expect(page", "page.goto", "page.click",
		},
		"cypress": {
			"cypress", "cy.visit", "cy.get", "cy.click", "cy.type", "describe(",
		},
		"selenium": {
			"selenium-webdriver", "webdriver", "driver.get", "driver.findElement",
		},
		"puppeteer": {
			"puppeteer", "browser.newPage", "page.goto", "page.click", "page.type",
		},
		"webdriverio": {
			"webdriverio", "browser.url", "browser.click", "$(",
		},
		"testcafe": {
			"testcafe", "fixture(", "test(", "Selector(", ".click()",
		},
		"nightwatch": {
			"nightwatch", ".url(", ".click(", ".setValue(",
		},
	}
	
	votes := make(map[string]int)
	
	for framework, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(lowerContent, keyword) {
				votes[framework]++
			}
		}
	}
	
	// Return framework with most matches
	var bestFramework string
	var maxVotes int
	
	for framework, count := range votes {
		if count > maxVotes {
			maxVotes = count
			bestFramework = framework
		}
	}
	
	return bestFramework
}

func (fd *FrameworkDetector) detectLanguage() string {
	// Check for TypeScript first
	tsFiles := []string{"tsconfig.json", "src/**/*.ts", "**/*.ts"}
	for _, pattern := range tsFiles {
		if len(fd.globFiles(pattern)) > 0 {
			return "typescript"
		}
	}
	
	// Check for Python
	pyFiles := []string{"requirements.txt", "setup.py", "**/*.py"}
	for _, pattern := range pyFiles {
		if len(fd.globFiles(pattern)) > 0 {
			return "python"
		}
	}
	
	// Default to JavaScript
	return "javascript"
}

func (fd *FrameworkDetector) inferRunCommand(scripts map[string]string, frameworkName string) string {
	// Common script names for E2E testing
	scriptPatterns := []string{
		"test:e2e", "e2e", "test:integration", "test:ui",
		frameworkName, "test:" + frameworkName,
	}
	
	for _, pattern := range scriptPatterns {
		if _, exists := scripts[pattern]; exists {
			return "npm run " + pattern
		}
	}
	
	return ""
}

func (fd *FrameworkDetector) enhanceWithConfigFiles(framework *E2EFramework) {
	// Try to find actual config file
	if framework.ConfigFile != "" {
		configPath := filepath.Join(fd.projectRoot, framework.ConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			// Config file exists, we're good
			return
		}
	}
	
	// Try to find alternative config files
	alternatives := fd.getConfigAlternatives(framework.Name)
	for _, alt := range alternatives {
		if _, err := os.Stat(filepath.Join(fd.projectRoot, alt)); err == nil {
			framework.ConfigFile = alt
			break
		}
	}
}

func (fd *FrameworkDetector) getConfigAlternatives(frameworkName string) []string {
	alternatives := map[string][]string{
		"playwright": {"playwright.config.js", "playwright.config.ts"},
		"cypress":    {"cypress.config.js", "cypress.config.ts", "cypress.json"},
		"webdriverio": {"wdio.conf.js", "wdio.conf.ts"},
	}
	
	return alternatives[frameworkName]
}

func (fd *FrameworkDetector) createFrameworkFromName(name string) *E2EFramework {
	frameworks := map[string]*E2EFramework{
		"playwright": {
			Name:        "playwright",
			DisplayName: "Playwright",
			RunCommand:  "npx playwright test",
		},
		"cypress": {
			Name:        "cypress",
			DisplayName: "Cypress",
			RunCommand:  "npx cypress run",
		},
		"selenium": {
			Name:        "selenium", 
			DisplayName: "Selenium WebDriver",
			RunCommand:  "npm test",
		},
		"puppeteer": {
			Name:        "puppeteer",
			DisplayName: "Puppeteer",
			RunCommand:  "npm test",
		},
		"webdriverio": {
			Name:        "webdriverio",
			DisplayName: "WebdriverIO",
			RunCommand:  "npx wdio run",
		},
		"testcafe": {
			Name:        "testcafe",
			DisplayName: "TestCafe",
			RunCommand:  "npx testcafe",
		},
		"nightwatch": {
			Name:        "nightwatch",
			DisplayName: "Nightwatch.js",
			RunCommand:  "npx nightwatch",
		},
	}
	
	if framework, exists := frameworks[name]; exists {
		framework.Language = fd.detectLanguage()
		return framework
	}
	
	return nil
}

func cleanVersion(version string) string {
	// Clean version strings like "^1.40.0" -> "1.40.0"
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, "<=")
	version = strings.TrimPrefix(version, ">")
	version = strings.TrimPrefix(version, "<")
	return strings.TrimSpace(version)
}

// DirectoryInfo represents a directory that might be created
type DirectoryInfo struct {
	Path        string `json:"path"`
	Purpose     string `json:"purpose"`
	Exists      bool   `json:"exists"`
	Required    bool   `json:"required"`
	Permissions string `json:"permissions"`
}

// EnsureTestDirectories ensures that test directories exist for the framework
func (fd *FrameworkDetector) EnsureTestDirectories(framework *E2EFramework, interactive bool) error {
	if framework.TestDir == "" {
		return nil // No test directory to create
	}

	dirs := fd.getRequiredDirectories(framework)
	
	// Check which directories need to be created
	var toCreate []DirectoryInfo
	for _, dir := range dirs {
		if !dir.Exists {
			toCreate = append(toCreate, dir)
		}
	}
	
	if len(toCreate) == 0 {
		return nil // All directories already exist
	}
	
	// If interactive, ask for confirmation
	if interactive {
		return fd.createDirectoriesInteractive(toCreate)
	}
	
	// Create directories without confirmation
	return fd.createDirectoriesNonInteractive(toCreate)
}

// getRequiredDirectories returns a list of directories needed for the framework
func (fd *FrameworkDetector) getRequiredDirectories(framework *E2EFramework) []DirectoryInfo {
	var dirs []DirectoryInfo
	
	// Main test directory
	testDirPath := filepath.Join(fd.projectRoot, framework.TestDir)
	dirs = append(dirs, DirectoryInfo{
		Path:        testDirPath,
		Purpose:     "Main test directory for " + framework.DisplayName + " tests",
		Exists:      fd.directoryExists(testDirPath),
		Required:    true,
		Permissions: "0755",
	})
	
	// Framework-specific subdirectories
	switch framework.Name {
	case "cypress":
		if framework.TestDir != "cypress" && !strings.Contains(framework.TestDir, "cypress") {
			// If test dir is not cypress-specific, add cypress subdirs
			supportDir := filepath.Join(testDirPath, "support")
			fixturesDir := filepath.Join(testDirPath, "fixtures")
			
			dirs = append(dirs,
				DirectoryInfo{
					Path:        supportDir,
					Purpose:     "Cypress support files and commands",
					Exists:      fd.directoryExists(supportDir),
					Required:    false,
					Permissions: "0755",
				},
				DirectoryInfo{
					Path:        fixturesDir,
					Purpose:     "Cypress test fixtures and data",
					Exists:      fd.directoryExists(fixturesDir),
					Required:    false,
					Permissions: "0755",
				},
			)
		}
		
	case "playwright":
		// Playwright might need utils or page objects
		utilsDir := filepath.Join(testDirPath, "utils")
		pagesDir := filepath.Join(testDirPath, "pages")
		
		dirs = append(dirs,
			DirectoryInfo{
				Path:        utilsDir,
				Purpose:     "Test utilities and helpers",
				Exists:      fd.directoryExists(utilsDir),
				Required:    false,
				Permissions: "0755",
			},
			DirectoryInfo{
				Path:        pagesDir,
				Purpose:     "Page Object Model classes",
				Exists:      fd.directoryExists(pagesDir),
				Required:    false,
				Permissions: "0755",
			},
		)
	}
	
	return dirs
}

// directoryExists checks if a directory exists
func (fd *FrameworkDetector) directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// createDirectoriesInteractive shows directory creation preview and asks for confirmation
func (fd *FrameworkDetector) createDirectoriesInteractive(toCreate []DirectoryInfo) error {
	fmt.Println("\n📁 The following directories will be created:")
	
	for _, dir := range toCreate {
		relPath := fd.getRelativePath(dir.Path)
		if dir.Required {
			fmt.Printf("   ✅ %s - %s\n", relPath, dir.Purpose)
		} else {
			fmt.Printf("   📂 %s - %s (optional)\n", relPath, dir.Purpose)
		}
	}
	
	// Create selector for user choice
	options := []components.SelectorOption{
		{
			ID:          "create_all",
			Title:       "Create all directories",
			Description: "Create both required and optional directories",
		},
		{
			ID:          "create_required",
			Title:       "Create required only",
			Description: "Create only the required test directory",
		},
		{
			ID:          "skip",
			Title:       "Skip directory creation",
			Description: "Don't create any directories now",
		},
	}
	
	choice, err := components.RunSelector("\n🤔 What would you like to do?", options)
	if err != nil {
		return err
	}
	
	switch choice {
	case "create_all":
		return fd.createDirectoriesNonInteractive(toCreate)
	case "create_required":
		var required []DirectoryInfo
		for _, dir := range toCreate {
			if dir.Required {
				required = append(required, dir)
			}
		}
		return fd.createDirectoriesNonInteractive(required)
	case "skip":
		fmt.Println("   ⏭️  Skipped directory creation")
		return nil
	}
	
	return nil
}

// createDirectoriesNonInteractive creates directories without user interaction
func (fd *FrameworkDetector) createDirectoriesNonInteractive(toCreate []DirectoryInfo) error {
	for _, dir := range toCreate {
		if err := os.MkdirAll(dir.Path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir.Path, err)
		}
		
		relPath := fd.getRelativePath(dir.Path)
		fmt.Printf("   ✅ Created %s\n", relPath)
	}
	
	return nil
}

// getRelativePath returns a path relative to the project root
func (fd *FrameworkDetector) getRelativePath(path string) string {
	relPath, err := filepath.Rel(fd.projectRoot, path)
	if err != nil {
		return path
	}
	return relPath
}