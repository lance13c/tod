package llm

import (
	"context"
	"log"
	"os"

	"github.com/lance13c/tod/internal/types"
)

// openAIClientSimple is a simplified implementation that delegates to mock
// TODO: This currently just delegates to the mock client and doesn't make real OpenAI API calls
// The mock client returns sample test code for test generation requests
type openAIClientSimple struct {
	apiKey   string
	model    string
	mock     *mockClient
	costCalc *CostCalculator
}

// newOpenAIClient creates a new OpenAI client - now using the real implementation
func newOpenAIClient(apiKey string, options map[string]interface{}) (Client, error) {
	// Use the real OpenAI client implementation
	return newRealOpenAIClient(apiKey, options)
}

// AnalyzeCode delegates to mock implementation
func (c *openAIClientSimple) AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error) {
	// Log the API call
	logFile, err := os.OpenFile(".tod/api_calls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "[OPENAI_CLIENT] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("AnalyzeCode called - delegating to mock client")
		logger.Printf("API Key present: %v", c.apiKey != "")
		logger.Printf("Model: %s", c.model)
		logger.Printf("FilePath: %s", filePath)
	}
	
	// Currently delegates to mock - this should make real OpenAI API calls
	result, err := c.mock.AnalyzeCode(ctx, code, filePath)
	
	if logFile != nil {
		logger := log.New(logFile, "[OPENAI_CLIENT] ", log.LstdFlags|log.Lmicroseconds)
		if err != nil {
			logger.Printf("Mock returned error: %v", err)
		} else {
			logger.Printf("Mock returned successfully, Notes length: %d", len(result.Notes))
		}
	}
	
	return result, err
}

// GenerateFlow delegates to mock implementation
func (c *openAIClientSimple) GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error) {
	return c.mock.GenerateFlow(ctx, actions)
}

// ExtractActions delegates to mock implementation
func (c *openAIClientSimple) ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error) {
	return c.mock.ExtractActions(ctx, code, framework, language)
}

// ResearchFramework delegates to mock implementation with enhanced results
func (c *openAIClientSimple) ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error) {
	research, err := c.mock.ResearchFramework(ctx, frameworkName, version)
	if err != nil {
		return nil, err
	}

	// Enhance with "OpenAI-powered" results
	research.Notes = "Enhanced research using OpenAI GPT-4 (simulated)"
	research.Confidence = research.Confidence * 1.05 // Slightly higher confidence
	if research.Confidence > 1.0 {
		research.Confidence = 1.0
	}

	return research, nil
}

// InterpretCommand implements the Client interface using the mock client
func (c *openAIClientSimple) InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error) {
	// For now, use the mock implementation but enhance it slightly for OpenAI
	interpretation, err := c.mock.InterpretCommand(ctx, command, availableActions)
	if err != nil {
		return nil, err
	}

	// Slightly enhance confidence for OpenAI
	interpretation.Confidence = interpretation.Confidence * 1.05
	if interpretation.Confidence > 1.0 {
		interpretation.Confidence = 1.0
	}

	return interpretation, nil
}

// AnalyzeScreenshot delegates to mock implementation
func (c *openAIClientSimple) AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error) {
	return c.mock.AnalyzeScreenshot(ctx, screenshot, prompt)
}

// GetLastUsage implements the Client interface
func (c *openAIClientSimple) GetLastUsage() *UsageStats {
	return c.mock.GetLastUsage()
}

// InterpretCommandWithContext implements the Client interface
func (c *openAIClientSimple) InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error) {
	// Delegate to regular interpret command - simple client doesn't need conversation context
	return c.InterpretCommand(ctx, command, availableActions)
}

// EstimateCost implements the Client interface
func (c *openAIClientSimple) EstimateCost(operation string, inputSize int) *UsageStats {
	// Get token estimates from mock (for token calculation logic)
	mockEstimate := c.mock.EstimateCost(operation, inputSize)
	
	// Use real cost calculator with provider and model
	return c.costCalc.EstimateCost("openai", c.model, mockEstimate.InputTokens, mockEstimate.OutputTokens)
}

// RankNavigationElements delegates to mock implementation
func (c *openAIClientSimple) RankNavigationElements(ctx context.Context, userInput string, elements []NavigationElement) (*NavigationRanking, error) {
	return c.mock.RankNavigationElements(ctx, userInput, elements)
}
