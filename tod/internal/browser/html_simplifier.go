package browser

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// SimplifyHTML simplifies HTML for LLM processing by removing unnecessary elements
// and keeping only test-relevant interactive elements and structure
func SimplifyHTML(htmlContent string) (string, error) {
	// Parse the HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Simplify the document
	simplifyNode(doc)

	// Remove empty text nodes and collapse whitespace
	cleanupNode(doc)

	// Render back to string
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return "", fmt.Errorf("failed to render HTML: %w", err)
	}

	return buf.String(), nil
}

// simplifyNode recursively simplifies HTML nodes
func simplifyNode(n *html.Node) {
	// Track nodes to remove
	var toRemove []*html.Node

	// Process child nodes first
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		simplifyNode(c)

		// Mark nodes for removal based on type and tag
		if shouldRemoveNode(c) {
			toRemove = append(toRemove, c)
		}
	}

	// Remove marked nodes
	for _, node := range toRemove {
		n.RemoveChild(node)
	}

	// Simplify attributes on remaining nodes
	if n.Type == html.ElementNode {
		simplifyAttributes(n)
	}
}

// shouldRemoveNode determines if a node should be removed from simplified HTML
func shouldRemoveNode(n *html.Node) bool {
	if n.Type == html.ElementNode {
		switch n.DataAtom {
		// Remove script, style, and meta elements
		case atom.Script, atom.Style, atom.Meta, atom.Link, atom.Noscript:
			return true
		// Keep SVG elements but simplify them
		case atom.Svg:
			// Replace with placeholder
			n.Data = "div"
			n.DataAtom = atom.Div
			n.Attr = []html.Attribute{{Key: "class", Val: "svg-placeholder"}}
			return false
		}

		// Remove hidden elements
		for _, attr := range n.Attr {
			if attr.Key == "style" && strings.Contains(attr.Val, "display:none") {
				return true
			}
			if attr.Key == "hidden" {
				return true
			}
		}
	}

	// Remove comment nodes
	if n.Type == html.CommentNode {
		return true
	}

	return false
}

// simplifyAttributes keeps only test-relevant attributes
func simplifyAttributes(n *html.Node) {
	var keepAttrs []html.Attribute

	// Define which attributes to keep for testing
	relevantAttrs := map[string]bool{
		"id":             true,
		"class":          true,
		"data-testid":    true,
		"data-test":      true,
		"data-cy":        true,
		"aria-label":     true,
		"aria-labelledby": true,
		"role":           true,
		"type":           true,
		"name":           true,
		"placeholder":    true,
		"href":           true,
		"action":         true,
		"method":         true,
		"value":          true,
		"checked":        true,
		"selected":       true,
		"disabled":       true,
		"readonly":       true,
		"required":       true,
	}

	// Keep only relevant attributes
	for _, attr := range n.Attr {
		// Always keep data-* attributes for testing
		if strings.HasPrefix(attr.Key, "data-") {
			keepAttrs = append(keepAttrs, attr)
		} else if relevantAttrs[attr.Key] {
			// Simplify class attribute to only keep first few classes
			if attr.Key == "class" {
				classes := strings.Fields(attr.Val)
				if len(classes) > 3 {
					attr.Val = strings.Join(classes[:3], " ") + " ..."
				}
			}
			keepAttrs = append(keepAttrs, attr)
		}
	}

	n.Attr = keepAttrs
}

// cleanupNode removes empty text nodes and normalizes whitespace
func cleanupNode(n *html.Node) {
	var toRemove []*html.Node

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		cleanupNode(c)

		if c.Type == html.TextNode {
			// Trim and normalize whitespace
			c.Data = strings.TrimSpace(c.Data)
			if c.Data == "" {
				toRemove = append(toRemove, c)
			} else {
				// Collapse multiple spaces
				c.Data = strings.Join(strings.Fields(c.Data), " ")
			}
		}
	}

	// Remove empty text nodes
	for _, node := range toRemove {
		n.RemoveChild(node)
	}
}

// ExtractInteractiveElements extracts a summary of interactive elements from HTML
func ExtractInteractiveElements(htmlContent string) ([]InteractiveElement, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var elements []InteractiveElement
	extractElements(doc, &elements)
	return elements, nil
}

// InteractiveElement represents an interactive HTML element
type InteractiveElement struct {
	Tag        string
	ID         string
	Class      string
	TestID     string
	Type       string
	Text       string
	AriaLabel  string
	Href       string
	Selector   string
	FullUrl    string // Full resolved URL for navigation
	Title      string // Title attribute
	IsNavigation bool // True if this is a navigation link
	IsButton   bool   // True if this is a button or button-like element
}

// extractElements recursively extracts interactive elements
func extractElements(n *html.Node, elements *[]InteractiveElement) {
	if n.Type == html.ElementNode {
		// Check if this is an interactive element
		if isInteractiveElement(n) {
			elem := InteractiveElement{
				Tag: n.Data,
			}

			// Extract attributes
			for _, attr := range n.Attr {
				switch attr.Key {
				case "id":
					elem.ID = attr.Val
				case "class":
					elem.Class = attr.Val
				case "data-testid", "data-test", "data-cy":
					elem.TestID = attr.Val
				case "type":
					elem.Type = attr.Val
				case "aria-label":
					elem.AriaLabel = attr.Val
				case "href":
					elem.Href = attr.Val
				}
			}

			// Extract text content
			elem.Text = extractText(n)

			// Build selector
			elem.Selector = buildSelector(elem)

			*elements = append(*elements, elem)
		}
	}

	// Process children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractElements(c, elements)
	}
}

// isInteractiveElement checks if a node is an interactive element
func isInteractiveElement(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Button, atom.A, atom.Input, atom.Select, atom.Textarea:
		return true
	case atom.Div, atom.Span:
		// Check for role="button" or onclick attributes
		for _, attr := range n.Attr {
			if attr.Key == "role" && attr.Val == "button" {
				return true
			}
			if strings.HasPrefix(attr.Key, "on") {
				return true
			}
		}
	}
	return false
}

// extractText extracts visible text from a node
func extractText(n *html.Node) string {
	var text strings.Builder
	extractTextHelper(n, &text)
	return strings.TrimSpace(text.String())
}

func extractTextHelper(n *html.Node, text *strings.Builder) {
	if n.Type == html.TextNode {
		text.WriteString(n.Data)
		text.WriteString(" ")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextHelper(c, text)
	}
}

// buildSelector builds a CSS selector for an element
func buildSelector(elem InteractiveElement) string {
	if elem.TestID != "" {
		return fmt.Sprintf("[data-testid='%s']", elem.TestID)
	}
	if elem.ID != "" {
		return fmt.Sprintf("#%s", elem.ID)
	}
	
	selector := elem.Tag
	if elem.Type != "" {
		selector += fmt.Sprintf("[type='%s']", elem.Type)
	}
	if elem.AriaLabel != "" {
		selector += fmt.Sprintf("[aria-label='%s']", elem.AriaLabel)
	}
	if selector == elem.Tag && elem.Text != "" {
		// Use text as last resort
		selector += fmt.Sprintf(":contains('%s')", strings.Split(elem.Text, " ")[0])
	}
	
	return selector
}