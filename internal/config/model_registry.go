package config

import (
	"fmt"
	"strings"
)

// ModelInfo contains information about a specific AI model
type ModelInfo struct {
	ID          string // Unique identifier (e.g., "gpt-5")
	DisplayName string // What to show in UI (e.g., "GPT-5 (OpenAI)")
	Provider    string // Provider identifier (openai, anthropic, google, etc.)
	ModelName   string // Actual model name to use in API calls
	Description string // Short description
}

// ModelRegistry holds all available models
var ModelRegistry = []ModelInfo{
	// OpenAI Models (2025)
	{
		ID:          "gpt-5",
		DisplayName: "GPT-5 (OpenAI)",
		Provider:    "openai",
		ModelName:   "gpt-5",
		Description: "Latest flagship model with enhanced reasoning",
	},
	{
		ID:          "gpt-4o",
		DisplayName: "GPT-4o (OpenAI)",
		Provider:    "openai",
		ModelName:   "gpt-4o",
		Description: "Omni multimodal model",
	},
	{
		ID:          "o3",
		DisplayName: "o3 (OpenAI)",
		Provider:    "openai",
		ModelName:   "o3",
		Description: "Advanced reasoning for complex problems",
	},
	{
		ID:          "o3-mini",
		DisplayName: "o3-mini (OpenAI)",
		Provider:    "openai",
		ModelName:   "o3-mini",
		Description: "Fast coding and math optimized",
	},
	{
		ID:          "gpt-4-turbo",
		DisplayName: "GPT-4 Turbo (OpenAI)",
		Provider:    "openai",
		ModelName:   "gpt-4-turbo",
		Description: "Previous generation high-performance model",
	},

	// Anthropic Claude Models (2025)
	{
		ID:          "claude-4-opus",
		DisplayName: "Claude 4 Opus (Anthropic)",
		Provider:    "anthropic",
		ModelName:   "claude-4-opus-20250601",
		Description: "Best for coding and complex reasoning",
	},
	{
		ID:          "claude-4-sonnet",
		DisplayName: "Claude 4 Sonnet (Anthropic)",
		Provider:    "anthropic",
		ModelName:   "claude-4-sonnet-20250601",
		Description: "Balanced performance and cost",
	},
	{
		ID:          "claude-3.5-sonnet",
		DisplayName: "Claude 3.5 Sonnet (Anthropic)",
		Provider:    "anthropic",
		ModelName:   "claude-3-5-sonnet-20241022",
		Description: "Proven reliable model for most tasks",
	},
	{
		ID:          "claude-3-opus",
		DisplayName: "Claude 3 Opus (Anthropic)",
		Provider:    "anthropic",
		ModelName:   "claude-3-opus-20240229",
		Description: "High-intelligence model",
	},

	// Google Gemini Models (2025)
	{
		ID:          "gemini-2.5-pro",
		DisplayName: "Gemini 2.5 Pro (Google)",
		Provider:    "google",
		ModelName:   "gemini-2.5-pro",
		Description: "Deep thinking mode with 2M token context",
	},
	{
		ID:          "gemini-2.5-flash",
		DisplayName: "Gemini 2.5 Flash (Google)",
		Provider:    "google",
		ModelName:   "gemini-2.5-flash",
		Description: "Fast and affordable with great performance",
	},
	{
		ID:          "gemini-2.0-flash",
		DisplayName: "Gemini 2.0 Flash (Google)",
		Provider:    "google",
		ModelName:   "gemini-2.0-flash-exp",
		Description: "Previous fast model, still excellent",
	},
	{
		ID:          "gemini-pro",
		DisplayName: "Gemini Pro (Google)",
		Provider:    "google",
		ModelName:   "gemini-pro",
		Description: "Reliable general-purpose model",
	},

	// Meta Llama Models (2025)
	{
		ID:          "llama-4-maverick",
		DisplayName: "Llama 4 Maverick (Meta)",
		Provider:    "meta",
		ModelName:   "llama-4-maverick-400b",
		Description: "400B open source powerhouse",
	},
	{
		ID:          "llama-4-scout",
		DisplayName: "Llama 4 Scout (Meta)",
		Provider:    "meta",
		ModelName:   "llama-4-scout-109b",
		Description: "109B efficient model with 512K context",
	},
	{
		ID:          "llama-3.3-70b",
		DisplayName: "Llama 3.3 70B (Meta)",
		Provider:    "openrouter",
		ModelName:   "meta-llama/llama-3.3-70b-instruct",
		Description: "Open source alternative via OpenRouter",
	},

	// xAI Grok Models (2025)
	{
		ID:          "grok-4-heavy",
		DisplayName: "Grok-4 Heavy (xAI)",
		Provider:    "xai",
		ModelName:   "grok-4-heavy",
		Description: "Most powerful multi-agent model, 50% on Humanity's Last Exam",
	},
	{
		ID:          "grok-4",
		DisplayName: "Grok-4 (xAI)",
		Provider:    "xai",
		ModelName:   "grok-4",
		Description: "Latest flagship model with tool use and multimodal capabilities",
	},
	{
		ID:          "grok-3",
		DisplayName: "Grok-3 (xAI)",
		Provider:    "xai",
		ModelName:   "grok-3",
		Description: "Truth-seeking AI with powerful reasoning",
	},
	{
		ID:          "grok-beta",
		DisplayName: "Grok Beta (xAI)",
		Provider:    "xai",
		ModelName:   "grok-beta",
		Description: "Previous generation Grok model",
	},

	// DeepSeek Models (2025)
	{
		ID:          "deepseek-r1",
		DisplayName: "DeepSeek R1 (DeepSeek)",
		Provider:    "openrouter",
		ModelName:   "deepseek/deepseek-r1:nitro",
		Description: "Advanced reasoning model via OpenRouter",
	},
	{
		ID:          "deepseek-coder",
		DisplayName: "DeepSeek Coder (DeepSeek)",
		Provider:    "openrouter",
		ModelName:   "deepseek/deepseek-coder",
		Description: "Specialized for code generation",
	},

	// Mistral/OpenRouter Models (2025)
	{
		ID:          "mixtral-8x22b",
		DisplayName: "Mixtral 8x22B (Mistral)",
		Provider:    "openrouter",
		ModelName:   "mistralai/mixtral-8x22b-instruct",
		Description: "Large mixture of experts model",
	},
	{
		ID:          "mistral-small",
		DisplayName: "Mistral Small 3.1 (Mistral)",
		Provider:    "openrouter",
		ModelName:   "mistralai/mistral-small-2412",
		Description: "24B efficient model",
	},
	{
		ID:          "qwen-coder",
		DisplayName: "Qwen 2.5 Coder 32B (Alibaba)",
		Provider:    "openrouter",
		ModelName:   "qwen/qwen-2.5-coder-32b-instruct",
		Description: "Specialized coding model",
	},
}

// GetModelByID returns a model by its ID
func GetModelByID(id string) (*ModelInfo, error) {
	for _, model := range ModelRegistry {
		if model.ID == id {
			return &model, nil
		}
	}
	return nil, fmt.Errorf("model not found: %s", id)
}

// ParseModelSelection parses a model selection string
// Supports:
// - "gpt-5" -> finds in registry
// - "custom:openrouter:some/model" -> creates custom model
func ParseModelSelection(selection string) (*ModelInfo, error) {
	// Handle custom models first
	if strings.HasPrefix(selection, "custom:") {
		parts := strings.SplitN(selection, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("custom model format should be 'custom:provider:model-name'")
		}
		
		return &ModelInfo{
			ID:          selection,
			DisplayName: fmt.Sprintf("%s (Custom %s)", parts[2], strings.Title(parts[1])),
			Provider:    parts[1],
			ModelName:   parts[2],
			Description: "Custom model",
		}, nil
	}

	// Try to find in registry
	model, err := GetModelByID(selection)
	if err != nil {
		return nil, fmt.Errorf("unknown model: %s. Use format 'custom:provider:model-name' for unlisted models", selection)
	}

	return model, nil
}

// GetModelSuggestions returns model suggestions for autocomplete
func GetModelSuggestions() []string {
	suggestions := make([]string, len(ModelRegistry))
	for i, model := range ModelRegistry {
		suggestions[i] = model.DisplayName
	}
	return suggestions
}

// GetModelIDs returns all model IDs for matching
func GetModelIDs() []string {
	ids := make([]string, len(ModelRegistry))
	for i, model := range ModelRegistry {
		ids[i] = model.ID
	}
	return ids
}

// FilterModelsByProvider returns models for a specific provider
func FilterModelsByProvider(provider string) []ModelInfo {
	var filtered []ModelInfo
	for _, model := range ModelRegistry {
		if model.Provider == provider {
			filtered = append(filtered, model)
		}
	}
	return filtered
}