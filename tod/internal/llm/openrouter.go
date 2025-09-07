package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ciciliostudio/tod/internal/types"
)

const (
	openRouterBaseURL        = "https://openrouter.ai/api/v1"
	openRouterModelsEndpoint = "/models"
)

// OpenRouterModel represents a model from OpenRouter API
type OpenRouterModel struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Pricing      OpenRouterPricing      `json:"pricing,omitempty"`
	Context      int                    `json:"context_length,omitempty"`
	Architecture OpenRouterArchitecture `json:"architecture,omitempty"`
	TopProvider  OpenRouterTopProvider  `json:"top_provider,omitempty"`
}

// OpenRouterPricing represents pricing information
type OpenRouterPricing struct {
	Prompt     string `json:"prompt,omitempty"`
	Completion string `json:"completion,omitempty"`
	Request    string `json:"request,omitempty"`
	Image      string `json:"image,omitempty"`
}

// OpenRouterArchitecture represents model architecture info
type OpenRouterArchitecture struct {
	Modality     string `json:"modality,omitempty"`
	Tokenizer    string `json:"tokenizer,omitempty"`
	InstructType string `json:"instruct_type,omitempty"`
}

// OpenRouterTopProvider represents provider info
type OpenRouterTopProvider struct {
	MaxCompletionTokens *int `json:"max_completion_tokens,omitempty"`
	IsModerated         bool `json:"is_moderated,omitempty"`
}

// OpenRouterModelsResponse represents the API response for models
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// OpenRouterClient implements the LLM Client interface for OpenRouter
type OpenRouterClient struct {
	apiKey       string
	model        string
	baseURL      string
	httpClient   *http.Client
	lastUsage    *UsageStats
	costCalc     *CostCalculator
	tokenCounter *TokenCounter
}

// newOpenRouterClient creates a new OpenRouter client
func newOpenRouterClient(apiKey string, options map[string]interface{}) (Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}

	model := "openai/gpt-4-turbo"
	if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	baseURL := openRouterBaseURL
	if url, ok := options["base_url"].(string); ok && url != "" {
		baseURL = url
	}

	return &OpenRouterClient{
		apiKey:       apiKey,
		model:        model,
		baseURL:      baseURL,
		httpClient:   &http.Client{},
		costCalc:     NewCostCalculator(),
		tokenCounter: NewTokenCounter(),
	}, nil
}

// FetchModels retrieves available models from OpenRouter API
func (c *OpenRouterClient) FetchModels(ctx context.Context) ([]OpenRouterModel, error) {
	url := c.baseURL + openRouterModelsEndpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/ciciliostudio/tod")
	req.Header.Set("X-Title", "Tod - Text-adventure Interface Framework")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResponse OpenRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return modelsResponse.Data, nil
}

// makeAPIRequest performs an API request to OpenRouter
func (c *OpenRouterClient) makeAPIRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/ciciliostudio/tod")
	req.Header.Set("X-Title", "Tod - Text-adventure Interface Framework")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// AnalyzeCode implements the Client interface
func (c *OpenRouterClient) AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error) {
	// Estimate token usage and cost
	inputTokens, outputTokens := c.tokenCounter.EstimateCodeAnalysisTokens(code, filePath, "detected", c.model)
	_ = c.costCalc.CalculateCost("openrouter", c.model, inputTokens, outputTokens)

	prompt := fmt.Sprintf(smartActionDiscoveryPrompt, "detected", filePath, code)

	payload := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.1,
		"max_tokens":  1000,
	}

	respBody, err := c.makeAPIRequest(ctx, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}

	// Parse token usage from response
	actualInputTokens, actualOutputTokens, usageErr := TokenUsageFromResponse("openrouter", respBody)
	if usageErr != nil {
		// Fall back to estimates if parsing fails
		actualInputTokens = inputTokens
		actualOutputTokens = outputTokens
	}

	// Calculate actual usage
	actualUsage := c.costCalc.CalculateCost("openrouter", c.model, actualInputTokens, actualOutputTokens)
	c.lastUsage = actualUsage

	// Parse OpenAI-style response
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	// Parse the JSON response from the model
	content := strings.TrimSpace(response.Choices[0].Message.Content)

	// Try to extract JSON from the response
	var analysis CodeAnalysis
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		// If direct parsing fails, try to extract JSON from markdown code block
		if start := strings.Index(content, "```json"); start != -1 {
			start += 7
			if end := strings.Index(content[start:], "```"); end != -1 {
				jsonStr := content[start : start+end]
				if err := json.Unmarshal([]byte(jsonStr), &analysis); err == nil {
					analysis.Usage = actualUsage
					return &analysis, nil
				}
			}
		}

		// Fallback: create a basic analysis
		return &CodeAnalysis{
			Endpoints: []EndpointInfo{
				{
					Path:        "/unknown",
					Method:      "GET",
					Description: "Analysis failed - using fallback",
					LineNumber:  1,
				},
			},
			Confidence: 0.1,
			Notes:      "Failed to parse LLM response: " + err.Error(),
			Usage:      actualUsage,
		}, nil
	}

	analysis.Usage = actualUsage
	return &analysis, nil
}

// GenerateFlow implements the Client interface
func (c *OpenRouterClient) GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error) {
	actionsJSON, err := json.Marshal(actions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal actions: %w", err)
	}

	prompt := fmt.Sprintf(flowGenerationPrompt, string(actionsJSON))

	payload := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
		"max_tokens":  1500,
	}

	respBody, err := c.makeAPIRequest(ctx, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)

	var flow FlowSuggestion
	if err := json.Unmarshal([]byte(content), &flow); err != nil {
		// Try to extract JSON from markdown code block
		if start := strings.Index(content, "```json"); start != -1 {
			start += 7
			if end := strings.Index(content[start:], "```"); end != -1 {
				jsonStr := content[start : start+end]
				if err := json.Unmarshal([]byte(jsonStr), &flow); err == nil {
					return &flow, nil
				}
			}
		}

		return nil, fmt.Errorf("failed to parse flow response: %w", err)
	}

	return &flow, nil
}

// ExtractActions implements the Client interface
func (c *OpenRouterClient) ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error) {
	prompt := fmt.Sprintf(smartActionDiscoveryPrompt, framework, "unknown", code)

	payload := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.1,
		"max_tokens":  2000,
	}

	respBody, err := c.makeAPIRequest(ctx, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)

	var actions []types.CodeAction
	if err := json.Unmarshal([]byte(content), &actions); err != nil {
		// Try to extract JSON from markdown code block
		if start := strings.Index(content, "```json"); start != -1 {
			start += 7
			if end := strings.Index(content[start:], "```"); end != -1 {
				jsonStr := content[start : start+end]
				if err := json.Unmarshal([]byte(jsonStr), &actions); err == nil {
					return actions, nil
				}
			}
		}

		return nil, fmt.Errorf("failed to parse actions response: %w", err)
	}

	return actions, nil
}

// ResearchFramework implements the Client interface
func (c *OpenRouterClient) ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error) {
	prompt := fmt.Sprintf(frameworkResearchPrompt, frameworkName, version, frameworkName, version)

	payload := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.1,
		"max_tokens":  2000,
	}

	respBody, err := c.makeAPIRequest(ctx, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)

	var research FrameworkResearch
	if err := json.Unmarshal([]byte(content), &research); err != nil {
		// Try to extract JSON from markdown code block
		if start := strings.Index(content, "```json"); start != -1 {
			start += 7
			if end := strings.Index(content[start:], "```"); end != -1 {
				jsonStr := content[start : start+end]
				if err := json.Unmarshal([]byte(jsonStr), &research); err == nil {
					return &research, nil
				}
			}
		}

		return nil, fmt.Errorf("failed to parse research response: %w", err)
	}

	return &research, nil
}

// InterpretCommand implements the Client interface
func (c *OpenRouterClient) InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error) {
	// For now, use fallback pattern matching since we don't have a simple text completion API
	// In a full implementation, you would create the appropriate API payload and use makeAPIRequest
	return c.fallbackInterpretCommand(command, availableActions), nil
}

// fallbackInterpretCommand provides simple pattern matching when LLM fails
func (c *OpenRouterClient) fallbackInterpretCommand(command string, availableActions []types.CodeAction) *CommandInterpretation {
	command = strings.ToLower(strings.TrimSpace(command))

	interpretation := &CommandInterpretation{
		Intent:      command,
		Confidence:  0.6,
		Parameters:  make(map[string]string),
		Suggestions: []string{},
	}

	// Simple pattern matching
	switch {
	case strings.Contains(command, "navigate") || strings.Contains(command, "go to") || strings.Contains(command, "visit"):
		interpretation.CommandType = "navigation"
		if strings.Contains(command, "homepage") || strings.Contains(command, "home") {
			interpretation.Parameters["page"] = "/"
		}
	case strings.Contains(command, "sign in") || strings.Contains(command, "login"):
		interpretation.CommandType = "authentication"
		interpretation.Parameters["action"] = "sign_in"
	default:
		interpretation.CommandType = "unknown"
		interpretation.Confidence = 0.3
	}

	return interpretation
}


// formatActionsForLLM formats available actions for LLM context
func formatActionsForLLM(actions []types.CodeAction) string {
	if len(actions) == 0 {
		return "No actions available"
	}

	var formatted []string
	for _, action := range actions {
		formatted = append(formatted, fmt.Sprintf("- %s: %s (ID: %s)", action.Name, action.Description, action.ID))
	}

	return strings.Join(formatted, "\n")
}

// AnalyzeScreenshot implements the Client interface
func (c *OpenRouterClient) AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error) {
	// For now, delegate to a mock implementation
	// In the future, this could use OpenRouter's vision-capable models
	mockClient := &mockClient{}
	return mockClient.AnalyzeScreenshot(ctx, screenshot, prompt)
}

// GetLastUsage implements the Client interface
func (c *OpenRouterClient) GetLastUsage() *UsageStats {
	return c.lastUsage
}

// InterpretCommandWithContext implements conversation-aware command interpretation for OpenRouter
func (c *OpenRouterClient) InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error) {
	// For now, enhance the fallback with conversation context
	// TODO: Implement actual API call with conversation history
	interpretation := c.fallbackInterpretCommand(command, availableActions)
	
	// Enhanced logic based on conversation context
	if conversation != nil && len(conversation.Messages) > 0 {
		// Analyze recent conversation for better context understanding
		recentMessages := conversation.Messages
		if len(recentMessages) > 3 {
			recentMessages = recentMessages[len(recentMessages)-3:] // Last 3 messages
		}

		// Build conversation context for better interpretation
		contextClues := []string{}
		for _, msg := range recentMessages {
			if msg.Role == "user" {
				contextClues = append(contextClues, strings.ToLower(msg.Content))
			}
		}

		// Apply context-aware enhancements
		for _, clue := range contextClues {
			if strings.Contains(clue, "fill") && interpretation.CommandType == "unknown" {
				if strings.Contains(command, "email") || strings.Contains(command, "password") {
					interpretation.CommandType = "form_input"
					interpretation.Confidence = 0.8
					interpretation.Parameters["field_type"] = "credential"
				}
			}
			
			if strings.Contains(clue, "navigate") && interpretation.CommandType == "unknown" {
				if strings.Contains(command, "button") || strings.Contains(command, "link") {
					interpretation.CommandType = "interaction"
					interpretation.Confidence = 0.75
					interpretation.Parameters["element_type"] = "clickable"
				}
			}
		}

		// Boost confidence for contextual understanding
		if interpretation.CommandType != "unknown" {
			interpretation.Confidence = min(1.0, interpretation.Confidence+0.15)
		}
	}

	return interpretation, nil
}

// EstimateCost implements the Client interface
func (c *OpenRouterClient) EstimateCost(operation string, inputSize int) *UsageStats {
	var inputTokens, outputTokens int64

	switch operation {
	case "analyze_code":
		// Estimate based on input size (rough approximation)
		// inputSize is bytes, ~4 characters per token for code
		inputTokens = int64(inputSize) / 4
		outputTokens = 400 // typical analysis response
	case "generate_flow":
		inputTokens = int64(inputSize) / 4
		outputTokens = 600 // typical flow response
	case "research_framework":
		inputTokens = 200   // base prompt
		outputTokens = 1000 // typical research response
	default:
		inputTokens = int64(inputSize) / 4
		outputTokens = 500 // generic estimate
	}

	return c.costCalc.EstimateCost("openrouter", c.model, inputTokens, outputTokens)
}

// RankNavigationElements implements navigation element ranking for OpenRouter
func (c *OpenRouterClient) RankNavigationElements(ctx context.Context, userInput string, elements []NavigationElement) (*NavigationRanking, error) {
	// For now, fall back to mock implementation
	// TODO: Implement actual OpenRouter API call for ranking
	mock := &mockClient{}
	return mock.RankNavigationElements(ctx, userInput, elements)
}
