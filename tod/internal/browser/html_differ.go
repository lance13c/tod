package browser

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// HTMLSnapshot represents a snapshot of HTML content at a specific time
type HTMLSnapshot struct {
	HTML      string
	Hash      string
	Timestamp int64
}

// HTMLDiffer tracks HTML changes over time
type HTMLDiffer struct {
	lastSnapshot *HTMLSnapshot
}

// NewHTMLDiffer creates a new HTML differ
func NewHTMLDiffer() *HTMLDiffer {
	return &HTMLDiffer{}
}

// HasChanged checks if the HTML has changed compared to the last snapshot
func (d *HTMLDiffer) HasChanged(html string, timestamp int64) (bool, *HTMLSnapshot) {
	// Create hash of the HTML content
	hash := d.hashHTML(html)
	
	snapshot := &HTMLSnapshot{
		HTML:      html,
		Hash:      hash,
		Timestamp: timestamp,
	}
	
	// If this is the first snapshot, consider it changed
	if d.lastSnapshot == nil {
		d.lastSnapshot = snapshot
		return true, snapshot
	}
	
	// Check if hash has changed
	changed := d.lastSnapshot.Hash != hash
	
	if changed {
		d.lastSnapshot = snapshot
	}
	
	return changed, snapshot
}

// GetChangedSections extracts sections of HTML that have changed
func (d *HTMLDiffer) GetChangedSections(oldHTML, newHTML string) string {
	// Simple implementation: if content length changed significantly,
	// extract the new content that wasn't in the old HTML
	if len(newHTML) <= len(oldHTML) {
		return "" // No new content
	}
	
	// For now, return the new content that appears to be added
	// This is a simplified approach - in production you might want 
	// more sophisticated diff algorithms
	if strings.Contains(newHTML, oldHTML) {
		// Extract what's new
		return d.extractNewContent(oldHTML, newHTML)
	}
	
	// If structure changed significantly, return all new HTML
	return newHTML
}

// hashHTML creates a hash of HTML content for comparison
func (d *HTMLDiffer) hashHTML(html string) string {
	// Normalize whitespace and create hash
	normalized := strings.ReplaceAll(html, "\n", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")
	// Remove extra spaces
	fields := strings.Fields(normalized)
	normalized = strings.Join(fields, " ")
	
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

// extractNewContent attempts to extract newly added content
func (d *HTMLDiffer) extractNewContent(oldHTML, newHTML string) string {
	// Simple heuristic: look for common patterns that indicate new content
	// This is a basic implementation - could be enhanced with proper DOM diffing
	
	// If new HTML is significantly longer, try to find the new parts
	if len(newHTML) > int(float64(len(oldHTML))*1.1) {
		// Look for new interactive elements that weren't there before
		newElements := []string{}
		
		// Check for new buttons, links, inputs that weren't in old HTML
		interactiveElements := []string{"<button", "<a ", "<input", "<select", "<textarea"}
		
		for _, element := range interactiveElements {
			oldCount := strings.Count(oldHTML, element)
			newCount := strings.Count(newHTML, element)
			
			if newCount > oldCount {
				// Extract examples of this new element type
				lines := strings.Split(newHTML, "\n")
				for _, line := range lines {
					if strings.Contains(line, element) && !strings.Contains(oldHTML, strings.TrimSpace(line)) {
						newElements = append(newElements, strings.TrimSpace(line))
						// Limit to avoid too much content
						if len(newElements) >= 10 {
							break
						}
					}
				}
			}
		}
		
		if len(newElements) > 0 {
			return strings.Join(newElements, "\n")
		}
	}
	
	// Fallback: return a portion of the new content
	if len(newHTML) > 1000 {
		return newHTML[len(oldHTML):len(oldHTML)+1000] + "..."
	}
	
	return newHTML[len(oldHTML):]
}

// Reset resets the differ state
func (d *HTMLDiffer) Reset() {
	d.lastSnapshot = nil
}