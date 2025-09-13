package browser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

// CleanHTML optimizes HTML using goquery for better performance and cleaner output
// This method is more aggressive in removing unnecessary elements while preserving
// essential content and interactive elements for LLM processing
func CleanHTML(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML with goquery: %w", err)
	}

	// Remove script, style, and other non-content elements
	doc.Find("script, style, noscript, iframe, svg, link, meta").Remove()

	// Remove comments by filtering the HTML nodes
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		if s.Nodes != nil && len(s.Nodes) > 0 {
			node := s.Nodes[0]
			// Remove comment nodes from children
			var toRemove []*html.Node
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.CommentNode {
					toRemove = append(toRemove, c)
				}
			}
			for _, n := range toRemove {
				node.RemoveChild(n)
			}
		}
	})

	// Remove hidden elements
	doc.Find("[hidden]").Remove()
	doc.Find("[style*='display:none']").Remove()
	doc.Find("[style*='display: none']").Remove()
	doc.Find("[style*='visibility:hidden']").Remove()
	doc.Find("[style*='visibility: hidden']").Remove()

	// Preserve only essential attributes
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		if s.Nodes == nil || len(s.Nodes) == 0 {
			return
		}
		node := s.Nodes[0]
		var preservedAttrs []html.Attribute

		// Define which attributes to keep
		for _, attr := range node.Attr {
			// Keep test IDs, data attributes, and semantic attributes
			if attr.Key == "data-testid" ||
				attr.Key == "data-test" ||
				attr.Key == "data-cy" ||
				attr.Key == "id" ||
				attr.Key == "class" ||
				strings.HasPrefix(attr.Key, "aria-") ||
				attr.Key == "role" ||
				attr.Key == "href" ||
				attr.Key == "src" ||
				attr.Key == "alt" ||
				attr.Key == "title" ||
				attr.Key == "type" ||
				attr.Key == "name" ||
				attr.Key == "value" ||
				attr.Key == "placeholder" ||
				attr.Key == "checked" ||
				attr.Key == "selected" ||
				attr.Key == "disabled" ||
				attr.Key == "readonly" ||
				attr.Key == "required" ||
				attr.Key == "action" ||
				attr.Key == "method" {
				// Simplify class attribute if too long
				if attr.Key == "class" {
					classes := strings.Fields(attr.Val)
					if len(classes) > 3 {
						attr.Val = strings.Join(classes[:3], " ")
					}
				}
				// Truncate src/href if they're data URLs or very long
				if (attr.Key == "src" || attr.Key == "href") && len(attr.Val) > 100 {
					if strings.HasPrefix(attr.Val, "data:") {
						attr.Val = "data:..."
					} else if strings.HasPrefix(attr.Val, "blob:") {
						attr.Val = "blob:..."
					} else if len(attr.Val) > 100 {
						attr.Val = attr.Val[:100] + "..."
					}
				}
				preservedAttrs = append(preservedAttrs, attr)
			}
		}
		node.Attr = preservedAttrs
	})

	// Normalize whitespace in text nodes
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		if s.Nodes == nil || len(s.Nodes) == 0 {
			return
		}
		node := s.Nodes[0]
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				// Collapse multiple spaces and trim
				c.Data = strings.Join(strings.Fields(c.Data), " ")
			}
		}
	})

	// Get the cleaned HTML
	cleanedHTML, err := doc.Html()
	if err != nil {
		return "", fmt.Errorf("failed to render cleaned HTML: %w", err)
	}

	return cleanedHTML, nil
}

// CleanHTMLMinimal provides the most aggressive HTML cleaning for minimal LLM context
// Only preserves interactive elements and their immediate context
func CleanHTMLMinimal(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Remove all non-essential elements
	doc.Find("script, style, noscript, iframe, svg, link, meta, img, video, audio, canvas, embed, object, param, source, track").Remove()

	// Build a minimal representation focused on interactive elements
	var result strings.Builder
	result.WriteString("<html><body>")

	// Find all interactive elements
	doc.Find("button, a, input, select, textarea, [role='button'], [onclick], [data-testid], [data-test]").Each(func(i int, s *goquery.Selection) {
		tagName := goquery.NodeName(s)
		result.WriteString("<")
		result.WriteString(tagName)

		// Add minimal attributes
		if id, exists := s.Attr("id"); exists {
			result.WriteString(fmt.Sprintf(` id="%s"`, id))
		}
		if testId, exists := s.Attr("data-testid"); exists {
			result.WriteString(fmt.Sprintf(` data-testid="%s"`, testId))
		} else if testId, exists := s.Attr("data-test"); exists {
			result.WriteString(fmt.Sprintf(` data-test="%s"`, testId))
		}
		if href, exists := s.Attr("href"); exists && !strings.HasPrefix(href, "javascript:") {
			if len(href) > 50 {
				href = href[:50] + "..."
			}
			result.WriteString(fmt.Sprintf(` href="%s"`, href))
		}
		if inputType, exists := s.Attr("type"); exists {
			result.WriteString(fmt.Sprintf(` type="%s"`, inputType))
		}
		if name, exists := s.Attr("name"); exists {
			result.WriteString(fmt.Sprintf(` name="%s"`, name))
		}
		if ariaLabel, exists := s.Attr("aria-label"); exists {
			result.WriteString(fmt.Sprintf(` aria-label="%s"`, ariaLabel))
		}
		if role, exists := s.Attr("role"); exists {
			result.WriteString(fmt.Sprintf(` role="%s"`, role))
		}
		if placeholder, exists := s.Attr("placeholder"); exists {
			result.WriteString(fmt.Sprintf(` placeholder="%s"`, placeholder))
		}

		result.WriteString(">")

		// Add text content if it's short
		text := strings.TrimSpace(s.Text())
		if len(text) > 0 && len(text) < 50 {
			result.WriteString(text)
		} else if len(text) >= 50 {
			result.WriteString(text[:47] + "...")
		}

		result.WriteString("</")
		result.WriteString(tagName)
		result.WriteString(">\n")
	})

	result.WriteString("</body></html>")
	return result.String(), nil
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