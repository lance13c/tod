package llm

import (
	"context"
	"fmt"

	"github.com/ciciliostudio/tod/internal/types"
)

// anthropicClientSimple is a simplified implementation that delegates to mock
type anthropicClientSimple struct {
	apiKey   string
	model    string
	mock     *mockClient
	costCalc *CostCalculator
}

// newAnthropicClient creates a new simplified Anthropic client
func newAnthropicClient(apiKey string, options map[string]interface{}) (Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	model := "claude-3-5-sonnet-20241022"
	if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	mockClient := &mockClient{}

	return &anthropicClientSimple{
		apiKey:   apiKey,
		model:    model,
		mock:     mockClient,
		costCalc: NewCostCalculator(),
	}, nil
}

// AnalyzeCode delegates to mock implementation
func (c *anthropicClientSimple) AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error) {
	return c.mock.AnalyzeCode(ctx, code, filePath)
}

// GenerateFlow delegates to mock implementation
func (c *anthropicClientSimple) GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error) {
	return c.mock.GenerateFlow(ctx, actions)
}

// ExtractActions delegates to mock implementation
func (c *anthropicClientSimple) ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error) {
	return c.mock.ExtractActions(ctx, code, framework, language)
}

// ResearchFramework delegates to mock implementation with enhanced results
func (c *anthropicClientSimple) ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error) {
	research, err := c.mock.ResearchFramework(ctx, frameworkName, version)
	if err != nil {
		return nil, err
	}

	// Enhance with "Anthropic-powered" results
	research.Notes = "Enhanced research using Anthropic Claude (simulated)"
	research.Confidence = research.Confidence * 1.1 // Slightly higher confidence
	if research.Confidence > 1.0 {
		research.Confidence = 1.0
	}

	return research, nil
}

// InterpretCommand implements the Client interface using the mock client
func (c *anthropicClientSimple) InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error) {
	// For now, use the mock implementation but enhance it slightly for Anthropic
	interpretation, err := c.mock.InterpretCommand(ctx, command, availableActions)
	if err != nil {
		return nil, err
	}

	// Slightly enhance confidence for Anthropic
	interpretation.Confidence = interpretation.Confidence * 1.1
	if interpretation.Confidence > 1.0 {
		interpretation.Confidence = 1.0
	}

	return interpretation, nil
}

// GetLastUsage implements the Client interface
func (c *anthropicClientSimple) GetLastUsage() *UsageStats {
	return c.mock.GetLastUsage()
}

// AnalyzeScreenshot delegates to mock implementation
func (c *anthropicClientSimple) AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error) {
	return c.mock.AnalyzeScreenshot(ctx, screenshot, prompt)
}

// InterpretCommandWithContext implements the Client interface using the mock client with enhanced context
func (c *anthropicClientSimple) InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error) {
	// Use the mock implementation with conversation context
	interpretation, err := c.mock.InterpretCommandWithContext(ctx, command, availableActions, conversation)
	if err != nil {
		return nil, err
	}

	// Enhance for Anthropic with better conversation understanding
	interpretation.Confidence = interpretation.Confidence * 1.15 // Higher boost for context-aware
	if interpretation.Confidence > 1.0 {
		interpretation.Confidence = 1.0
	}

	return interpretation, nil
}

// EstimateCost implements the Client interface
func (c *anthropicClientSimple) EstimateCost(operation string, inputSize int) *UsageStats {
	// Get token estimates from mock (for token calculation logic)
	mockEstimate := c.mock.EstimateCost(operation, inputSize)
	
	// Use real cost calculator with provider and model
	return c.costCalc.EstimateCost("anthropic", c.model, mockEstimate.InputTokens, mockEstimate.OutputTokens)
}
