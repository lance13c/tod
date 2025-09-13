package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lance13c/tod/internal/types"
)

// OpenAIRequest represents the request structure for OpenAI API
type OpenAIRequest struct {
	Model                string          `json:"model"`
	Messages             []OpenAIMessage `json:"messages"`
	Temperature          *float64        `json:"temperature,omitempty"`
	MaxCompletionTokens  *int            `json:"max_completion_tokens,omitempty"`
}

// OpenAIMessage represents a message in the OpenAI format
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
	Error   *OpenAIError   `json:"error,omitempty"`
}

// OpenAIChoice represents a completion choice
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage information
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an error from the OpenAI API
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// openAIClient is the real OpenAI API client implementation
type openAIClient struct {
	apiKey      string
	model       string
	baseURL     string
	httpClient  *http.Client
	costCalc    *CostCalculator
	lastUsage   *UsageStats
	temperature float64
	maxTokens   int
}

// newRealOpenAIClient creates a new OpenAI client that makes real API calls
func newRealOpenAIClient(apiKey string, options map[string]interface{}) (Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	model := "gpt-4o-mini" // Default to GPT-4o-mini for cost efficiency
	if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	baseURL := "https://api.openai.com/v1"
	if url, ok := options["base_url"].(string); ok && url != "" {
		baseURL = url
	}

	temperature := 0.7
	if tempVal, exists := options["temperature"]; exists {
		switch v := tempVal.(type) {
		case float64:
			temperature = v
		case float32:
			temperature = float64(v)
		case int:
			temperature = float64(v)
		case int64:
			temperature = float64(v)
		case string:
			// Try to parse string as float
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				temperature = parsed
			}
		}
		// Ensure temperature is within valid range for OpenAI (0.0 to 2.0)
		if temperature < 0.0 {
			temperature = 0.0
		} else if temperature > 2.0 {
			temperature = 2.0
		}
	}

	maxTokens := 2000
	if maxVal, exists := options["max_tokens"]; exists {
		switch v := maxVal.(type) {
		case int:
			maxTokens = v
		case int64:
			maxTokens = int(v)
		case float64:
			maxTokens = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				maxTokens = parsed
			}
		}
	}

	return &openAIClient{
		apiKey: apiKey,
		model:  model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // Increased to 3 minutes for larger requests
		},
		costCalc:    NewCostCalculator(),
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// makeRequest makes a request to the OpenAI API
func (c *openAIClient) makeRequest(ctx context.Context, messages []OpenAIMessage) (*OpenAIResponse, error) {
	// Log the API call
	logFile, _ := os.OpenFile(".tod/api_calls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logFile != nil {
		defer logFile.Close()
		logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Making API request to OpenAI")
		logger.Printf("Model: %s, Temperature: %.2f, MaxTokens: %d", c.model, c.temperature, c.maxTokens)
		logger.Printf("Number of messages: %d", len(messages))
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if len(lastMsg.Content) > 500 {
				logger.Printf("Last message preview (first 500 chars): %s...", lastMsg.Content[:500])
			} else {
				logger.Printf("Last message: %s", lastMsg.Content)
			}
		}
	}

	request := OpenAIRequest{
		Model:    c.model,
		Messages: messages,
	}
	
	// Don't send temperature - let the API use its default
	// Some models don't support custom temperature values
	
	// Only add max tokens if specified and > 0
	if c.maxTokens > 0 {
		request.MaxCompletionTokens = &c.maxTokens
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Log the actual JSON being sent for debugging
	if logFile != nil {
		logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Request JSON: %s", string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if logFile != nil {
			logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("ERROR: HTTP request failed: %v", err)
		}
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if logFile != nil {
		logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Response received in %v", duration)
		logger.Printf("Status: %d", resp.StatusCode)
		logger.Printf("Response length: %d bytes", len(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		if logFile != nil {
			logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("ERROR: Failed to parse response: %v", err)
			logger.Printf("Raw response: %s", string(body))
		}
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API errors
	if openAIResp.Error != nil {
		if logFile != nil {
			logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("ERROR from OpenAI API: %s", openAIResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	// Update usage stats
	if openAIResp.Usage.TotalTokens > 0 {
		costStats := c.costCalc.CalculateCost("openai", c.model, int64(openAIResp.Usage.PromptTokens), int64(openAIResp.Usage.CompletionTokens))
		c.lastUsage = &UsageStats{
			Provider:     "openai",
			Model:        c.model,
			InputTokens:  int64(openAIResp.Usage.PromptTokens),
			OutputTokens: int64(openAIResp.Usage.CompletionTokens),
			TotalTokens:  int64(openAIResp.Usage.TotalTokens),
			InputCost:    costStats.InputCost,
			OutputCost:   costStats.OutputCost,
			TotalCost:    costStats.TotalCost,
			RequestTime:  time.Now(),
		}
		
		if logFile != nil {
			logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Token usage - Prompt: %d, Completion: %d, Total: %d", 
				openAIResp.Usage.PromptTokens, openAIResp.Usage.CompletionTokens, openAIResp.Usage.TotalTokens)
			logger.Printf("Estimated cost: $%.4f", c.lastUsage.TotalCost)
		}
	}

	return &openAIResp, nil
}

// AnalyzeCode implements the Client interface with real OpenAI API calls
func (c *openAIClient) AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error) {
	// Ensure we have a valid context
	if ctx == nil {
		ctx = context.Background()
	}
	
	// For test generation, we need to format the prompt appropriately
	var systemPrompt string
	var userPrompt string

	if filePath == "test-generation.txt" {
		// This is a test generation request
		systemPrompt = "You are an expert test automation engineer. Generate clean, well-structured test code based on the provided requirements. Return ONLY the test code without any markdown formatting or explanations."
		userPrompt = code
	} else if filePath == "action-code.js" {
		// This is an action code generation request for browser automation
		systemPrompt = "You are an expert browser automation engineer. Generate executable JavaScript code for browser automation. Return your response exactly as requested in the prompt."
		userPrompt = code
	} else {
		// Regular code analysis
		systemPrompt = "You are an expert code analyst. Analyze the provided code and identify endpoints, authentication methods, and dependencies."
		userPrompt = fmt.Sprintf("Analyze this code from %s:\n\n%s", filePath, code)
	}

	messages := []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := c.makeRequest(ctx, messages)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI - empty choices array")
	}

	content := resp.Choices[0].Message.Content
	
	// Check for empty content
	if content == "" {
		// Log for debugging
		logFile, _ := os.OpenFile(".tod/api_calls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if logFile != nil {
			defer logFile.Close()
			logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("ERROR: Received empty content from OpenAI")
			logger.Printf("Model: %s, Finish reason: %s", c.model, resp.Choices[0].FinishReason)
			if resp.Usage.TotalTokens > 0 {
				logger.Printf("Tokens used - Input: %d, Output: %d", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
			}
		}
		return nil, fmt.Errorf("OpenAI returned empty response - model: %s, finish_reason: %s", c.model, resp.Choices[0].FinishReason)
	}

	// Log the response
	logFile, _ := os.OpenFile(".tod/api_calls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logFile != nil {
		defer logFile.Close()
		logger := log.New(logFile, "[OPENAI_REAL] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Response content length: %d", len(content))
		if len(content) > 500 {
			logger.Printf("Response preview (first 500 chars): %s...", content[:500])
		} else {
			logger.Printf("Full response: %s", content)
		}
	}

	// For test generation, return the content as Notes
	if filePath == "test-generation.txt" {
		return &CodeAnalysis{
			Notes:      content,
			Confidence: 0.95,
			Usage:      c.lastUsage,
		}, nil
	}

	// For regular analysis, try to parse structured data
	analysis := &CodeAnalysis{
		Endpoints:    []EndpointInfo{},
		AuthMethods:  []string{},
		Dependencies: []string{},
		Notes:        content,
		Confidence:   0.9,
		Usage:        c.lastUsage,
	}

	// Try to extract endpoints from the response
	if strings.Contains(content, "endpoint") || strings.Contains(content, "route") {
		// Basic extraction - could be enhanced with better parsing
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "POST") || strings.Contains(line, "GET") ||
				strings.Contains(line, "PUT") || strings.Contains(line, "DELETE") {
				// Found a potential endpoint mention
				endpoint := EndpointInfo{
					Path:        "/api/endpoint",
					Method:      "POST",
					Description: line,
					Parameters:  make(map[string]Param),
					Responses:   make(map[string]interface{}),
				}
				analysis.Endpoints = append(analysis.Endpoints, endpoint)
				break
			}
		}
	}

	return analysis, nil
}

// Other interface methods - delegate to simplified implementations for now
func (c *openAIClient) GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error) {
	// Use the existing mock implementation for now
	mock := &mockClient{}
	return mock.GenerateFlow(ctx, actions)
}

func (c *openAIClient) ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error) {
	mock := &mockClient{}
	return mock.ExtractActions(ctx, code, framework, language)
}

func (c *openAIClient) ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error) {
	mock := &mockClient{}
	return mock.ResearchFramework(ctx, frameworkName, version)
}

func (c *openAIClient) InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error) {
	mock := &mockClient{}
	return mock.InterpretCommand(ctx, command, availableActions)
}

func (c *openAIClient) InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error) {
	return c.InterpretCommand(ctx, command, availableActions)
}

func (c *openAIClient) AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error) {
	mock := &mockClient{}
	return mock.AnalyzeScreenshot(ctx, screenshot, prompt)
}

func (c *openAIClient) GetLastUsage() *UsageStats {
	return c.lastUsage
}

func (c *openAIClient) EstimateCost(operation string, inputSize int) *UsageStats {
	// Rough estimation based on input size
	estimatedTokens := inputSize / 4 // Approximate token count
	costStats := c.costCalc.CalculateCost("openai", c.model, int64(estimatedTokens), 500)
	return &UsageStats{
		Provider:     "openai",
		Model:        c.model,
		InputTokens:  int64(estimatedTokens),
		OutputTokens: 500,
		TotalTokens:  int64(estimatedTokens + 500),
		InputCost:    costStats.InputCost,
		OutputCost:   costStats.OutputCost,
		TotalCost:    costStats.TotalCost,
		RequestTime:  time.Now(),
	}
}

// RankNavigationElements implements navigation element ranking using OpenAI
func (c *openAIClient) RankNavigationElements(ctx context.Context, userInput string, elements []NavigationElement) (*NavigationRanking, error) {
	// Format elements for the prompt
	elementsText := ""
	for i, element := range elements {
		elementsText += fmt.Sprintf("%d. Text: \"%s\", Selector: \"%s\", URL: \"%s\", Type: \"%s\"\n", 
			i+1, element.Text, element.Selector, element.URL, element.Type)
	}

	// Create the prompt
	prompt := fmt.Sprintf(navigationRankingPrompt, userInput, elementsText)

	// Make the API call
	messages := []OpenAIMessage{
		{Role: "system", Content: "You are a web navigation expert. Return only valid JSON."},
		{Role: "user", Content: prompt},
	}

	resp, err := c.makeRequest(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	// Parse the JSON response
	var rankedElements []RankedNavigationElement
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &rankedElements); err != nil {
		// If JSON parsing fails, fall back to mock implementation
		log.Printf("Failed to parse OpenAI navigation ranking response, falling back to basic matching: %v", err)
		mock := &mockClient{}
		return mock.RankNavigationElements(ctx, userInput, elements)
	}

	return &NavigationRanking{
		Elements: rankedElements,
		Usage:    c.lastUsage,
	}, nil
}