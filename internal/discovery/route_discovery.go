package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RouteDiscovery handles intelligent discovery of route and page directories
type RouteDiscovery struct {
	projectRoot string
}

// DiscoveredDirectory represents a discovered directory containing routes/pages
type DiscoveredDirectory struct {
	Path        string   `json:"path"`
	Type        string   `json:"type"`        // "pages", "routes", "api", "components"
	Framework   string   `json:"framework"`   // "nextjs", "express", "fastapi", etc.
	FileCount   int      `json:"file_count"`
	Extensions  []string `json:"extensions"`
	Confidence  float64  `json:"confidence"`  // 0.0 to 1.0
	Description string   `json:"description"`
}

// NewRouteDiscovery creates a new route discovery instance
func NewRouteDiscovery(projectRoot string) *RouteDiscovery {
	return &RouteDiscovery{
		projectRoot: projectRoot,
	}
}

// DiscoverRouteDirectories finds directories likely to contain routes, pages, and APIs
func (rd *RouteDiscovery) DiscoverRouteDirectories() ([]DiscoveredDirectory, error) {
	var discovered []DiscoveredDirectory
	
	// Define patterns for different frameworks and architectures
	patterns := []struct {
		glob        string
		dirType     string
		framework   string
		confidence  float64
		description string
	}{
		// Next.js patterns
		{"app", "pages", "nextjs", 0.95, "Next.js App Router directory"},
		{"pages", "pages", "nextjs", 0.9, "Next.js Pages Router directory"},
		{"pages/api", "api", "nextjs", 0.95, "Next.js API Routes"},
		{"app/api", "api", "nextjs", 0.95, "Next.js App Router API"},
		{"src/app", "pages", "nextjs", 0.9, "Next.js App Router in src"},
		{"src/pages", "pages", "nextjs", 0.85, "Next.js Pages Router in src"},
		
		// React patterns
		{"src/pages", "pages", "react", 0.8, "React pages directory"},
		{"src/routes", "routes", "react", 0.8, "React routes directory"},
		{"src/components/pages", "pages", "react", 0.7, "React page components"},
		
		// Express.js patterns
		{"routes", "routes", "express", 0.85, "Express.js routes directory"},
		{"api", "api", "express", 0.8, "Express.js API routes"},
		{"src/routes", "routes", "express", 0.8, "Express.js routes in src"},
		{"controllers", "routes", "express", 0.75, "Express.js controllers"},
		
		// FastAPI patterns
		{"routers", "routes", "fastapi", 0.85, "FastAPI routers directory"},
		{"api/routers", "routes", "fastapi", 0.9, "FastAPI API routers"},
		{"app/routers", "routes", "fastapi", 0.85, "FastAPI app routers"},
		
		// Generic patterns
		{"views", "pages", "generic", 0.6, "Generic views directory"},
		{"endpoints", "api", "generic", 0.7, "Generic endpoints directory"},
		{"handlers", "routes", "generic", 0.65, "Generic handlers directory"},
	}
	
	for _, pattern := range patterns {
		fullPath := filepath.Join(rd.projectRoot, pattern.glob)
		
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			fileCount, extensions := rd.analyzeDirectory(fullPath)
			
			if fileCount > 0 {
				discovered = append(discovered, DiscoveredDirectory{
					Path:        pattern.glob,
					Type:        pattern.dirType,
					Framework:   pattern.framework,
					FileCount:   fileCount,
					Extensions:  extensions,
					Confidence:  pattern.confidence,
					Description: pattern.description,
				})
			}
		}
	}
	
	// Sort by confidence (highest first)
	for i := 0; i < len(discovered); i++ {
		for j := i + 1; j < len(discovered); j++ {
			if discovered[j].Confidence > discovered[i].Confidence {
				discovered[i], discovered[j] = discovered[j], discovered[i]
			}
		}
	}
	
	return discovered, nil
}

// analyzeDirectory counts files and determines extensions in a directory
func (rd *RouteDiscovery) analyzeDirectory(dirPath string) (int, []string) {
	fileCount := 0
	extMap := make(map[string]bool)
	
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		
		// Skip common ignore patterns
		if strings.Contains(path, "node_modules") || 
		   strings.Contains(path, ".git") ||
		   strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		
		ext := filepath.Ext(path)
		if rd.isRelevantExtension(ext) {
			fileCount++
			extMap[ext] = true
		}
		
		return nil
	})
	
	// Convert map to slice
	extensions := make([]string, 0, len(extMap))
	for ext := range extMap {
		extensions = append(extensions, ext)
	}
	
	return fileCount, extensions
}

// isRelevantExtension checks if a file extension is relevant for route/page discovery
func (rd *RouteDiscovery) isRelevantExtension(ext string) bool {
	relevantExts := []string{
		".js", ".jsx", ".ts", ".tsx",  // JavaScript/TypeScript
		".py",                         // Python
		".go",                         // Go
		".java", ".kt",               // JVM
		".rs",                        // Rust
		".php",                       // PHP
		".rb",                        // Ruby
		".vue", ".svelte",            // Component frameworks
	}
	
	ext = strings.ToLower(ext)
	for _, relevant := range relevantExts {
		if ext == relevant {
			return true
		}
	}
	return false
}

// GetRecommendedDirectories returns the most likely directories for analysis
func (rd *RouteDiscovery) GetRecommendedDirectories() ([]string, error) {
	discovered, err := rd.DiscoverRouteDirectories()
	if err != nil {
		return nil, err
	}
	
	var recommended []string
	seen := make(map[string]bool)
	
	// Add high-confidence directories
	for _, dir := range discovered {
		if dir.Confidence >= 0.8 && !seen[dir.Path] {
			recommended = append(recommended, dir.Path)
			seen[dir.Path] = true
		}
	}
	
	// If no high-confidence directories found, add medium-confidence ones
	if len(recommended) == 0 {
		for _, dir := range discovered {
			if dir.Confidence >= 0.6 && !seen[dir.Path] {
				recommended = append(recommended, dir.Path)
				seen[dir.Path] = true
			}
		}
	}
	
	return recommended, nil
}

// FormatDiscoveryResults formats the discovery results for display
func (rd *RouteDiscovery) FormatDiscoveryResults(discovered []DiscoveredDirectory) string {
	if len(discovered) == 0 {
		return "No route/page directories found"
	}
	
	var result strings.Builder
	result.WriteString("Discovered route/page directories:\n")
	
	for i, dir := range discovered {
		if i >= 5 { // Limit to top 5 results
			result.WriteString(fmt.Sprintf("   ... and %d more\n", len(discovered)-5))
			break
		}
		
		confidence := int(dir.Confidence * 100)
		result.WriteString(fmt.Sprintf("   %s %s (%d files, %d%% confidence)\n", 
			rd.getTypeEmoji(dir.Type), dir.Path, dir.FileCount, confidence))
		result.WriteString(fmt.Sprintf("     %s\n", dir.Description))
	}
	
	return result.String()
}

// getTypeEmoji returns an emoji for the directory type
func (rd *RouteDiscovery) getTypeEmoji(dirType string) string {
	switch dirType {
	case "pages":
		return "📄"
	case "routes":
		return "🛣️"
	case "api":
		return "🔗"
	case "components":
		return "🧩"
	default:
		return "📁"
	}
}