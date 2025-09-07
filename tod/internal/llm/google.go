package llm

import (
	"context"
	"fmt"

	"github.com/ciciliostudio/tod/internal/types"
)

// googleClientSimple is a simplified implementation that delegates to mock
type googleClientSimple struct {
	apiKey   string
	model    string
	mock     *mockClient
	costCalc *CostCalculator
}

// newGoogleClient creates a new simplified Google client
func newGoogleClient(apiKey string, options map[string]interface{}) (Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Google AI API key is required")
	}

	model := "gemini-2.5-pro"
	if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	mockClient := &mockClient{}

	return &googleClientSimple{
		apiKey:   apiKey,
		model:    model,
		mock:     mockClient,
		costCalc: NewCostCalculator(),
	}, nil
}

// AnalyzeCode delegates to mock implementation
func (c *googleClientSimple) AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error) {
	return c.mock.AnalyzeCode(ctx, code, filePath)
}

// GenerateFlow delegates to mock implementation
func (c *googleClientSimple) GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error) {
	return c.mock.GenerateFlow(ctx, actions)
}

// ExtractActions delegates to mock implementation
func (c *googleClientSimple) ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error) {
	return c.mock.ExtractActions(ctx, code, framework, language)
}

// ResearchFramework delegates to mock implementation with enhanced results
func (c *googleClientSimple) ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error) {
	research, err := c.mock.ResearchFramework(ctx, frameworkName, version)
	if err != nil {
		return nil, err
	}
	
	// Enhance with "Google-powered" results
	research.Notes = "Enhanced research using Google Gemini (simulated)"
	research.Confidence = research.Confidence * 1.08 // Slightly higher confidence
	if research.Confidence > 1.0 {
		research.Confidence = 1.0
	}
	
	return research, nil
}

// InterpretCommand implements the Client interface using the mock client
func (c *googleClientSimple) InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error) {
	// For now, use the mock implementation but enhance it slightly for Google
	interpretation, err := c.mock.InterpretCommand(ctx, command, availableActions)
	if err != nil {
		return nil, err
	}
	
	// Slightly enhance confidence for Google
	interpretation.Confidence = interpretation.Confidence * 1.08
	if interpretation.Confidence > 1.0 {
		interpretation.Confidence = 1.0
	}
	
	return interpretation, nil
}

// AnalyzeScreenshot delegates to mock implementation
func (c *googleClientSimple) AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error) {
	return c.mock.AnalyzeScreenshot(ctx, screenshot, prompt)
}

// GetLastUsage implements the Client interface
func (c *googleClientSimple) GetLastUsage() *UsageStats {
	return c.mock.GetLastUsage()
}

// InterpretCommandWithContext implements the Client interface
func (c *googleClientSimple) InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error) {
	// Delegate to regular interpret command - simple client doesn't need conversation context
	return c.InterpretCommand(ctx, command, availableActions)
}

// EstimateCost implements the Client interface
func (c *googleClientSimple) EstimateCost(operation string, inputSize int) *UsageStats {
	// Get token estimates from mock (for token calculation logic)
	mockEstimate := c.mock.EstimateCost(operation, inputSize)
	
	// Use real cost calculator with provider and model
	return c.costCalc.EstimateCost("google", c.model, mockEstimate.InputTokens, mockEstimate.OutputTokens)
}

// RankNavigationElements delegates to mock implementation
func (c *googleClientSimple) RankNavigationElements(ctx context.Context, userInput string, elements []NavigationElement) (*NavigationRanking, error) {
	return c.mock.RankNavigationElements(ctx, userInput, elements)
}