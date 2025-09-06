package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// TokenCounter provides token counting functionality for different models
type TokenCounter struct {
	// Cache for token counts to avoid recalculating
	cache map[string]int
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		cache: make(map[string]int),
	}
}

// CountTokens estimates token count for a given text and model
// This is a simplified approximation - for production use, integrate with tiktoken or similar
func (tc *TokenCounter) CountTokens(text, model string) int64 {
	// Create cache key
	cacheKey := fmt.Sprintf("%s:%s", model, text)

	// Check cache first
	if count, exists := tc.cache[cacheKey]; exists {
		return int64(count)
	}

	// Estimate tokens based on model family
	var count int
	switch {
	case strings.Contains(strings.ToLower(model), "gpt"):
		count = tc.estimateGPTTokens(text)
	case strings.Contains(strings.ToLower(model), "claude"):
		count = tc.estimateClaudeTokens(text)
	default:
		// Default to GPT-style counting
		count = tc.estimateGPTTokens(text)
	}

	// Cache the result
	tc.cache[cacheKey] = count

	return int64(count)
}

// estimateGPTTokens provides a rough estimate for GPT models
// Real implementation should use tiktoken library
func (tc *TokenCounter) estimateGPTTokens(text string) int {
	// GPT models roughly follow these patterns:
	// - 1 token ≈ 4 characters in English
	// - 1 token ≈ ¾ words in English
	// - 100 tokens ≈ 75 words

	// Simple approximation: count words and multiply by 1.33
	words := tc.countWords(text)
	tokens := float64(words) * 1.33

	// Add some overhead for special tokens, formatting, etc.
	overhead := float64(len(text)) * 0.05

	return int(tokens + overhead)
}

// estimateClaudeTokens provides a rough estimate for Claude models
// Claude uses a similar tokenizer to GPT models
func (tc *TokenCounter) estimateClaudeTokens(text string) int {
	// Claude tokenization is similar to GPT
	// Use the same estimation with slight adjustment
	return int(float64(tc.estimateGPTTokens(text)) * 1.05)
}

// countWords counts words in text
func (tc *TokenCounter) countWords(text string) int {
	if text == "" {
		return 0
	}

	// Split by whitespace and count non-empty parts
	words := strings.Fields(text)
	return len(words)
}

// EstimatePromptTokens estimates tokens for a structured prompt
func (tc *TokenCounter) EstimatePromptTokens(prompt StructuredPrompt, model string) int64 {
	totalText := ""

	if prompt.System != "" {
		totalText += prompt.System + "\n\n"
	}

	if prompt.User != "" {
		totalText += prompt.User + "\n\n"
	}

	if prompt.Context != "" {
		totalText += prompt.Context + "\n\n"
	}

	return tc.CountTokens(totalText, model)
}

// EstimateCodeAnalysisTokens estimates tokens for code analysis requests
func (tc *TokenCounter) EstimateCodeAnalysisTokens(code, filePath, framework string, model string) (int64, int64) {
	// Input tokens: system prompt + user prompt + code
	systemPrompt := fmt.Sprintf(smartActionDiscoveryPrompt, framework, filePath, code)
	inputTokens := tc.CountTokens(systemPrompt, model)

	// Output tokens: estimated based on typical response size
	// Code analysis typically returns 200-800 tokens depending on complexity
	codeLength := utf8.RuneCountInString(code)
	var estimatedOutputTokens int64

	switch {
	case codeLength < 500:
		estimatedOutputTokens = 200 // Small files
	case codeLength < 2000:
		estimatedOutputTokens = 400 // Medium files
	case codeLength < 5000:
		estimatedOutputTokens = 600 // Large files
	default:
		estimatedOutputTokens = 800 // Very large files
	}

	return inputTokens, estimatedOutputTokens
}

// EstimateFlowGenerationTokens estimates tokens for flow generation
func (tc *TokenCounter) EstimateFlowGenerationTokens(actions []string, model string) (int64, int64) {
	// Convert actions to JSON-like string for estimation
	actionsText := ""
	for _, action := range actions {
		actionsText += action + "\n"
	}

	prompt := fmt.Sprintf(flowGenerationPrompt, actionsText)
	inputTokens := tc.CountTokens(prompt, model)

	// Flow generation typically returns 300-1200 tokens
	actionCount := len(actions)
	var estimatedOutputTokens int64

	switch {
	case actionCount <= 2:
		estimatedOutputTokens = 300
	case actionCount <= 5:
		estimatedOutputTokens = 600
	case actionCount <= 10:
		estimatedOutputTokens = 900
	default:
		estimatedOutputTokens = 1200
	}

	return inputTokens, estimatedOutputTokens
}

// EstimateFrameworkResearchTokens estimates tokens for framework research
func (tc *TokenCounter) EstimateFrameworkResearchTokens(framework, version string, model string) (int64, int64) {
	prompt := fmt.Sprintf(frameworkResearchPrompt, framework, version, framework, version)
	inputTokens := tc.CountTokens(prompt, model)

	// Framework research typically returns 800-1500 tokens
	estimatedOutputTokens := int64(1000)

	return inputTokens, estimatedOutputTokens
}

// StructuredPrompt represents a structured LLM prompt
type StructuredPrompt struct {
	System  string
	User    string
	Context string
}

// ClearCache clears the token counting cache
func (tc *TokenCounter) ClearCache() {
	tc.cache = make(map[string]int)
}

// GetCacheSize returns the current cache size
func (tc *TokenCounter) GetCacheSize() int {
	return len(tc.cache)
}

// TokenUsageFromResponse extracts token usage from API response
// This should be implemented per provider based on their response format
func TokenUsageFromResponse(provider string, responseBody []byte) (inputTokens, outputTokens int64, err error) {
	switch provider {
	case "openai", "openrouter":
		return parseOpenAIUsage(responseBody)
	case "anthropic":
		return parseAnthropicUsage(responseBody)
	default:
		return 0, 0, fmt.Errorf("unsupported provider for token usage parsing: %s", provider)
	}
}

// parseOpenAIUsage parses OpenAI API response for token usage
func parseOpenAIUsage(responseBody []byte) (inputTokens, outputTokens int64, err error) {
	var response struct {
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return 0, 0, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	return response.Usage.PromptTokens, response.Usage.CompletionTokens, nil
}

// parseAnthropicUsage parses Anthropic API response for token usage
func parseAnthropicUsage(responseBody []byte) (inputTokens, outputTokens int64, err error) {
	var response struct {
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return 0, 0, fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	return response.Usage.InputTokens, response.Usage.OutputTokens, nil
}

// EstimateBatchAnalysisTokens estimates tokens for analyzing multiple files
func (tc *TokenCounter) EstimateBatchAnalysisTokens(filePaths []string, model string) (totalInputTokens, totalOutputTokens int64, fileDetails []BatchFileEstimate, err error) {
	fileDetails = make([]BatchFileEstimate, 0, len(filePaths))
	
	for _, filePath := range filePaths {
		// Read file content
		content, readErr := os.ReadFile(filePath)
		if readErr != nil {
			// Skip unreadable files but track them
			fileDetails = append(fileDetails, BatchFileEstimate{
				Path:         filePath,
				InputTokens:  0,
				OutputTokens: 0,
				Error:        readErr.Error(),
			})
			continue
		}
		
		// Estimate tokens for this file
		inputTokens, outputTokens := tc.EstimateCodeAnalysisTokens(string(content), filePath, "detected", model)
		
		fileDetails = append(fileDetails, BatchFileEstimate{
			Path:         filePath,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			FileSize:     len(content),
		})
		
		totalInputTokens += inputTokens
		totalOutputTokens += outputTokens
	}
	
	return totalInputTokens, totalOutputTokens, fileDetails, nil
}

// BatchFileEstimate represents token estimation for a single file
type BatchFileEstimate struct {
	Path         string `json:"path"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	FileSize     int    `json:"file_size"`
	Error        string `json:"error,omitempty"`
}

// FormatBatchEstimate formats a batch analysis estimate for display
func FormatBatchEstimate(inputTokens, outputTokens int64, cost float64, fileCount int) string {
	return fmt.Sprintf(`📊 Analysis Cost Estimate:
   Files to analyze: %d files
   Input tokens: %s
   Output tokens: %s
   Total tokens: %s
   Estimated cost: %s`,
		fileCount,
		FormatTokens(inputTokens),
		FormatTokens(outputTokens),
		FormatTokens(inputTokens+outputTokens),
		FormatCost(cost))
}
