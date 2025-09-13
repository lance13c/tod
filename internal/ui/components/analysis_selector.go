package components

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lance13c/tod/internal/discovery"
	"github.com/lance13c/tod/internal/llm"
)

// AnalysisOption represents an analysis choice
type AnalysisOption struct {
	ID          string
	Name        string
	Description string
}

// AnalysisChoice represents the user's analysis selection
type AnalysisChoice struct {
	Type        string   // "full", "partial", "smart", "skip"
	Directories []string // Selected directories for analysis
	EstimateCost bool    // Whether to show cost estimate
}

// CostEstimate represents a pre-calculated cost estimate
type CostEstimate struct {
	FileCount    int     `json:"file_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Cost         float64 `json:"cost"`
	ErrorMsg     string  `json:"error_msg,omitempty"`
}

// RunAnalysisSelector presents analysis options and returns the user's choice
func RunAnalysisSelector(projectRoot string, client llm.Client) (*AnalysisChoice, error) {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("\n🤖 Would you like Tod to analyze your codebase?")
	fmt.Println()
	
	// Discover route directories
	routeDiscovery := discovery.NewRouteDiscovery(projectRoot)
	discovered, err := routeDiscovery.DiscoverRouteDirectories()
	if err != nil {
		return nil, fmt.Errorf("failed to discover routes: %w", err)
	}
	
	// Show discovered directories
	if len(discovered) > 0 {
		fmt.Println(routeDiscovery.FormatDiscoveryResults(discovered))
		fmt.Println()
	}
	
	// Pre-calculate costs for smart and full analysis
	var smartEstimate, fullEstimate *CostEstimate
	recommended, _ := routeDiscovery.GetRecommendedDirectories()
	
	fmt.Println("💰 Calculating cost estimates...")
	
	if len(recommended) > 0 {
		smartEstimate = calculateCostEstimate(projectRoot, recommended, client)
	}
	
	fullEstimate = calculateCostEstimate(projectRoot, []string{"."}, client)
	
	fmt.Println("\nOptions:")
	
	// Option 1: Smart discovery
	if len(recommended) > 0 && smartEstimate != nil {
		if smartEstimate.ErrorMsg != "" {
			fmt.Printf("   1. Smart discovery - Auto-analyze detected directories\n")
			fmt.Printf("      📂 Directories: %s\n", strings.Join(recommended, ", "))
			fmt.Printf("      Cost estimation unavailable: %s\n", smartEstimate.ErrorMsg)
		} else {
			costStr := formatCost(smartEstimate.Cost)
			fmt.Printf("   1. Smart discovery - Auto-analyze detected directories (~%s)\n", costStr)
			fmt.Printf("      📂 Will analyze: %s (%d files)\n", strings.Join(recommended, ", "), smartEstimate.FileCount)
		}
	} else {
		fmt.Printf("   1. Smart discovery - Auto-analyze detected directories\n")
		fmt.Printf("      ❌ No route directories detected\n")
	}
	
	// Option 2: Partial analysis  
	fmt.Printf("   2. Partial analysis - Select specific directories\n")
	fmt.Printf("      💡 You choose which directories to analyze\n")
	
	// Option 3: Full analysis
	if fullEstimate != nil && fullEstimate.ErrorMsg == "" {
		costStr := formatCost(fullEstimate.Cost)
		fmt.Printf("   3. Full analysis - Analyze entire project (~%s)\n", costStr)
		fmt.Printf("      📁 %d files across all directories\n", fullEstimate.FileCount)
	} else {
		fmt.Printf("   3. Full analysis - Analyze entire project\n")
		fmt.Printf("      May be expensive - cost estimation unavailable\n")
	}
	
	// Option 4: Skip
	fmt.Printf("   4. Skip analysis - Configure manually later\n")
	fmt.Printf("      ℹ️  No AI costs, manual setup required\n")
	
	fmt.Println()
	
	// Get user choice
	choice := askChoice(reader, "Select option [1-4]: ", 1, 4)
	
	switch choice {
	case 1: // Smart discovery
		if len(recommended) == 0 {
			fmt.Println("❌ No route directories found. Falling back to manual selection.")
			return handlePartialAnalysisWithClient(reader, projectRoot, client)
		}
		
		// If cost is significant, ask for confirmation
		if smartEstimate != nil && smartEstimate.Cost > 0.05 {
			if !askYesNo(reader, "💸 Continue with analysis? (Y/n): ", true) {
				return handlePartialAnalysisWithClient(reader, projectRoot, client)
			}
		}
		
		return &AnalysisChoice{
			Type:         "smart",
			Directories:  recommended,
			EstimateCost: true,
		}, nil
		
	case 2: // Partial analysis
		return handlePartialAnalysisWithClient(reader, projectRoot, client)
		
	case 3: // Full analysis
		// Double-check for expensive analysis
		if fullEstimate != nil && fullEstimate.Cost > 0.1 {
			if !askYesNo(reader, "This will be expensive. Continue? (y/N): ", false) {
				return RunAnalysisSelector(projectRoot, client)
			}
		}
		
		return &AnalysisChoice{
			Type:         "full",
			Directories:  []string{"."},
			EstimateCost: true,
		}, nil
		
	case 4: // Skip analysis
		fmt.Println("\nSkipping AI analysis. You'll need to:")
		fmt.Println("   • Manually configure route patterns in .tod/config.yaml")
		fmt.Println("   • Define action mappings in .tod/manifest.json")
		fmt.Println("   • Or run 'tod analyze' later when ready")
		
		return &AnalysisChoice{
			Type:        "skip",
			Directories: []string{},
		}, nil
	}
	
	return nil, fmt.Errorf("invalid selection")
}

// handlePartialAnalysis handles partial directory selection with @ symbols
func handlePartialAnalysis(reader *bufio.Reader, projectRoot string) (*AnalysisChoice, error) {
	return handlePartialAnalysisWithClient(reader, projectRoot, nil)
}

// handlePartialAnalysisWithClient handles partial directory selection with @ symbols and cost estimation
func handlePartialAnalysisWithClient(reader *bufio.Reader, projectRoot string, client llm.Client) (*AnalysisChoice, error) {
	fmt.Println("\n📂 Partial Analysis - Select directories to analyze")
	fmt.Println("   Enter directories separated by spaces (use @ prefix)")
	fmt.Println("   Examples: @pages @api, @src/routes, @app")
	fmt.Println()
	
	// Show available directories as suggestions
	routeDiscovery := discovery.NewRouteDiscovery(projectRoot)
	discovered, _ := routeDiscovery.DiscoverRouteDirectories()
	
	if len(discovered) > 0 {
		fmt.Println("💡 Suggestions based on your project:")
		for i, dir := range discovered {
			if i >= 8 { // Limit suggestions
				break
			}
			fmt.Printf("   @%s (%d files)\n", dir.Path, dir.FileCount)
		}
		fmt.Println()
	}
	
	fmt.Print("Directories to analyze: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" {
		return &AnalysisChoice{
			Type:        "skip",
			Directories: []string{},
		}, nil
	}
	
	// Parse @ prefixed directories
	directories := parseDirectorySelection(input)
	
	if len(directories) == 0 {
		fmt.Println("❌ No valid directories specified")
		return handlePartialAnalysisWithClient(reader, projectRoot, client)
	}
	
	fmt.Printf("📂 Will analyze: %s\n", strings.Join(directories, ", "))
	
	// Show cost estimate if client is available
	if client != nil {
		if !showCostEstimate(projectRoot, directories, client) {
			// User declined after seeing cost
			fmt.Println("❌ Analysis cancelled")
			return &AnalysisChoice{
				Type:        "skip",
				Directories: []string{},
			}, nil
		}
	}
	
	return &AnalysisChoice{
		Type:         "partial",
		Directories:  directories,
		EstimateCost: true,
	}, nil
}

// parseDirectorySelection parses @ prefixed directory selection
func parseDirectorySelection(input string) []string {
	var directories []string
	parts := strings.Fields(input)
	
	for _, part := range parts {
		// Remove @ prefix if present
		dir := strings.TrimPrefix(part, "@")
		if dir != "" && dir != part { // Only add if @ was present
			directories = append(directories, dir)
		}
	}
	
	return directories
}

// Helper functions

func askChoice(reader *bufio.Reader, prompt string, min, max int) int {
	for {
		fmt.Printf("%s", prompt)
		
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		
		choice, err := strconv.Atoi(input)
		if err == nil && choice >= min && choice <= max {
			return choice
		}
		
		fmt.Printf("Please enter a number between %d and %d\n", min, max)
	}
}

func askYesNo(reader *bufio.Reader, prompt string, defaultVal bool) bool {
	fmt.Printf("%s", prompt)
	
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	
	if input == "" {
		return defaultVal
	}
	
	return input == "y" || input == "yes"
}

// formatCost formats a cost value for display
func formatCost(cost float64) string {
	if cost < 0.001 {
		return "< $0.001"
	} else if cost < 0.01 {
		return fmt.Sprintf("$%.3f", cost)
	} else if cost < 1.0 {
		return fmt.Sprintf("$%.2f", cost)
	} else {
		return fmt.Sprintf("$%.1f", cost)
	}
}

// calculateCostEstimate calculates cost estimate for analyzing specified directories
func calculateCostEstimate(projectRoot string, directories []string, client llm.Client) *CostEstimate {
	if client == nil {
		return &CostEstimate{ErrorMsg: "no client available"}
	}
	
	// Collect all files from directories
	var allFiles []string
	for _, dir := range directories {
		dirPath := filepath.Join(projectRoot, dir)
		files, err := collectRelevantFiles(dirPath)
		if err != nil {
			return &CostEstimate{ErrorMsg: fmt.Sprintf("error scanning %s: %v", dir, err)}
		}
		allFiles = append(allFiles, files...)
	}
	
	if len(allFiles) == 0 {
		return &CostEstimate{FileCount: 0}
	}
	
	// Create token counter and estimate costs
	tokenCounter := llm.NewTokenCounter()
	model := "gpt-4o-mini" // Default model for estimation
	
	inputTokens, outputTokens, _, err := tokenCounter.EstimateBatchAnalysisTokens(allFiles, model)
	if err != nil {
		return &CostEstimate{ErrorMsg: fmt.Sprintf("cost calculation error: %v", err)}
	}
	
	// Calculate cost (rough OpenAI pricing)
	inputCost := float64(inputTokens) * 0.00015 / 1000
	outputCost := float64(outputTokens) * 0.0006 / 1000
	totalCost := inputCost + outputCost
	
	return &CostEstimate{
		FileCount:    len(allFiles),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Cost:         totalCost,
	}
}

// showCostEstimate displays cost estimate for analyzing specified directories
func showCostEstimate(projectRoot string, directories []string, client llm.Client) bool {
	fmt.Println("\n💰 Calculating cost estimate...")
	
	// Collect all files from directories
	var allFiles []string
	for _, dir := range directories {
		dirPath := filepath.Join(projectRoot, dir)
		files, err := collectRelevantFiles(dirPath)
		if err != nil {
			fmt.Printf("⚠️ Could not scan directory %s: %v\n", dir, err)
			continue
		}
		allFiles = append(allFiles, files...)
	}
	
	if len(allFiles) == 0 {
		fmt.Println("❌ No files found to analyze")
		return false
	}
	
	// Create token counter and estimate costs
	tokenCounter := llm.NewTokenCounter()
	
	// For cost estimation, we'll assume a default model (can be made configurable)
	model := "gpt-4o-mini"
	
	inputTokens, outputTokens, _, err := tokenCounter.EstimateBatchAnalysisTokens(allFiles, model)
	if err != nil {
		fmt.Printf("⚠️ Could not estimate costs: %v\n", err)
		return true // Continue anyway
	}
	
	// Calculate cost (this should use the actual client's cost calculator)
	// For now, use rough OpenAI pricing: $0.15/1M input tokens, $0.60/1M output tokens
	inputCost := float64(inputTokens) * 0.00015 / 1000
	outputCost := float64(outputTokens) * 0.0006 / 1000
	totalCost := inputCost + outputCost
	
	// Display the estimate
	fmt.Printf("\n%s\n", llm.FormatBatchEstimate(inputTokens, outputTokens, totalCost, len(allFiles)))
	
	// Ask for confirmation if cost is significant
	if totalCost > 0.05 { // More than 5 cents
		return askYesNo(bufio.NewReader(os.Stdin), "💸 Continue with analysis? (y/N): ", false)
	}
	
	// Auto-approve for small costs
	fmt.Println("💚 Cost is minimal, proceeding automatically")
	return true
}

// collectRelevantFiles recursively finds relevant source files in a directory
func collectRelevantFiles(dirPath string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		if info.IsDir() {
			// Skip common ignore directories
			if shouldSkipDirectory(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this is a relevant source file
		if isRelevantSourceFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// shouldSkipDirectory determines if a directory should be skipped during file collection
func shouldSkipDirectory(dirName string) bool {
	skipDirs := []string{
		"node_modules", ".git", "dist", "build", ".next",
		"coverage", "__pycache__", ".pytest_cache", "vendor",
		".venv", "venv", "target", ".cargo",
	}

	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}
	return false
}

// isRelevantSourceFile determines if a file should be analyzed
func isRelevantSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	relevantExts := []string{
		".js", ".jsx", ".ts", ".tsx",  // JavaScript/TypeScript
		".py",                         // Python
		".go",                         // Go
		".java", ".kt",               // JVM languages
		".rs",                        // Rust
		".php",                       // PHP
		".rb",                        // Ruby
		".cs",                        // C#
		".vue", ".svelte",            // Component frameworks
	}

	for _, relevantExt := range relevantExts {
		if ext == relevantExt {
			return true
		}
	}
	return false
}