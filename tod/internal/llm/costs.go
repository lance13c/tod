package llm

import (
	"fmt"
	"strings"
	"time"
)

// ModelPricing represents pricing information for a specific model
type ModelPricing struct {
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	InputCost   float64   `json:"input_cost"`  // Cost per 1M input tokens in USD
	OutputCost  float64   `json:"output_cost"` // Cost per 1M output tokens in USD
	LastUpdated time.Time `json:"last_updated"`
}

// UsageStats represents the result of a single LLM request
type UsageStats struct {
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	InputCost    float64   `json:"input_cost"`
	OutputCost   float64   `json:"output_cost"`
	TotalCost    float64   `json:"total_cost"`
	RequestTime  time.Time `json:"request_time"`
}

// CostCalculator handles model pricing and cost calculations
type CostCalculator struct {
	pricing map[string]ModelPricing
}

// NewCostCalculator creates a new cost calculator with current model pricing
func NewCostCalculator() *CostCalculator {
	calc := &CostCalculator{
		pricing: make(map[string]ModelPricing),
	}
	calc.loadModelPricing()
	return calc
}

// loadModelPricing initializes the pricing database with current 2025 rates
func (c *CostCalculator) loadModelPricing() {
	now := time.Now()

	// OpenAI Models (as of 2025)
	c.pricing["openai/gpt-4o"] = ModelPricing{
		Provider:    "openai",
		Model:       "gpt-4o",
		InputCost:   3.00,  // $3 per 1M input tokens
		OutputCost:  10.00, // $10 per 1M output tokens
		LastUpdated: now,
	}

	c.pricing["openai/gpt-4o-mini"] = ModelPricing{
		Provider:    "openai",
		Model:       "gpt-4o-mini",
		InputCost:   0.15, // $0.15 per 1M input tokens
		OutputCost:  0.60, // $0.60 per 1M output tokens
		LastUpdated: now,
	}

	c.pricing["openai/gpt-4-turbo"] = ModelPricing{
		Provider:    "openai",
		Model:       "gpt-4-turbo",
		InputCost:   10.00, // $10 per 1M input tokens
		OutputCost:  30.00, // $30 per 1M output tokens
		LastUpdated: now,
	}

	c.pricing["openai/gpt-5"] = ModelPricing{
		Provider:    "openai",
		Model:       "gpt-5",
		InputCost:   5.00,  // $5 per 1M input tokens (estimated)
		OutputCost:  15.00, // $15 per 1M output tokens (estimated)
		LastUpdated: now,
	}

	// Anthropic Claude Models (as of 2025)
	c.pricing["anthropic/claude-3-5-sonnet"] = ModelPricing{
		Provider:    "anthropic",
		Model:       "claude-3-5-sonnet",
		InputCost:   3.00,  // $3 per 1M input tokens
		OutputCost:  15.00, // $15 per 1M output tokens
		LastUpdated: now,
	}

	c.pricing["anthropic/claude-3-haiku"] = ModelPricing{
		Provider:    "anthropic",
		Model:       "claude-3-haiku",
		InputCost:   0.25, // $0.25 per 1M input tokens
		OutputCost:  1.25, // $1.25 per 1M output tokens
		LastUpdated: now,
	}

	c.pricing["anthropic/claude-3-opus"] = ModelPricing{
		Provider:    "anthropic",
		Model:       "claude-3-opus",
		InputCost:   15.00, // $15 per 1M input tokens
		OutputCost:  75.00, // $75 per 1M output tokens
		LastUpdated: now,
	}

	// OpenRouter models use their own routing
	// Default to GPT-4o pricing for unknown OpenRouter models
	c.pricing["openrouter/default"] = ModelPricing{
		Provider:    "openrouter",
		Model:       "default",
		InputCost:   3.00,  // Conservative estimate
		OutputCost:  10.00, // Conservative estimate
		LastUpdated: now,
	}
}

// CalculateCost calculates the cost for a given usage
func (c *CostCalculator) CalculateCost(provider, model string, inputTokens, outputTokens int64) *UsageStats {
	pricing := c.getPricingForModel(provider, model)

	// Calculate costs (pricing is per 1M tokens)
	inputCost := (float64(inputTokens) / 1000000.0) * pricing.InputCost
	outputCost := (float64(outputTokens) / 1000000.0) * pricing.OutputCost
	totalCost := inputCost + outputCost

	return &UsageStats{
		Provider:     provider,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
		RequestTime:  time.Now(),
	}
}

// EstimateCost estimates the cost for a given input without making the request
func (c *CostCalculator) EstimateCost(provider, model string, estimatedInputTokens, estimatedOutputTokens int64) *UsageStats {
	return c.CalculateCost(provider, model, estimatedInputTokens, estimatedOutputTokens)
}

// getPricingForModel gets pricing for a specific model
func (c *CostCalculator) getPricingForModel(provider, model string) ModelPricing {
	// Try exact match first
	key := fmt.Sprintf("%s/%s", provider, model)
	if pricing, exists := c.pricing[key]; exists {
		return pricing
	}

	// Try provider-specific fallbacks
	switch provider {
	case "openai":
		// Default to GPT-4o for unknown OpenAI models
		if pricing, exists := c.pricing["openai/gpt-4o"]; exists {
			return pricing
		}
	case "anthropic":
		// Default to Claude-3.5-Sonnet for unknown Anthropic models
		if pricing, exists := c.pricing["anthropic/claude-3-5-sonnet"]; exists {
			return pricing
		}
	case "openrouter":
		// Check if it's a known model through OpenRouter
		modelKey := strings.ToLower(model)
		if strings.Contains(modelKey, "gpt-4") {
			return c.pricing["openai/gpt-4o"]
		}
		if strings.Contains(modelKey, "claude") {
			return c.pricing["anthropic/claude-3-5-sonnet"]
		}
		// Use OpenRouter default
		return c.pricing["openrouter/default"]
	}

	// Final fallback - use most expensive model as conservative estimate
	return ModelPricing{
		Provider:    provider,
		Model:       model,
		InputCost:   15.00, // Conservative high estimate
		OutputCost:  75.00, // Conservative high estimate
		LastUpdated: time.Now(),
	}
}

// GetModelPricing returns pricing information for a model
func (c *CostCalculator) GetModelPricing(provider, model string) ModelPricing {
	return c.getPricingForModel(provider, model)
}

// ListAvailablePricing returns all available model pricing
func (c *CostCalculator) ListAvailablePricing() []ModelPricing {
	var prices []ModelPricing
	for _, pricing := range c.pricing {
		prices = append(prices, pricing)
	}
	return prices
}

// FormatCost formats a cost in USD for display
func FormatCost(cost float64) string {
	if cost < 0.001 {
		return fmt.Sprintf("$%.4f", cost)
	} else if cost < 0.01 {
		return fmt.Sprintf("$%.3f", cost)
	} else {
		return fmt.Sprintf("$%.2f", cost)
	}
}

// FormatTokens formats token counts for display
func FormatTokens(tokens int64) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	} else if tokens < 1000000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000.0)
	} else {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000.0)
	}
}
