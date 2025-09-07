package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/agent/core"
	"github.com/ciciliostudio/tod/internal/config"
)

// FlowService provides high-level flow operations
type FlowService struct {
	agent       core.FlowAgent
	executor    *core.FlowExecutor
	storage     *FlowStorage
	config      *config.Config
	userConfig  *config.TestUserConfig
	projectRoot string
}

// FlowStorage handles flow persistence and caching
type FlowStorage struct {
	flows       map[string]*core.Flow
	lastUpdated time.Time
}

// NewFlowStorage creates a new flow storage
func NewFlowStorage() *FlowStorage {
	return &FlowStorage{
		flows: make(map[string]*core.Flow),
	}
}

// NewFlowService creates a new flow service
func NewFlowService(cfg *config.Config, projectRoot string) (*FlowService, error) {
	// Create agent
	agent, err := core.NewFlowAgent(cfg, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow agent: %w", err)
	}

	// Load user config
	userLoader := config.NewTestUserLoader(projectRoot)
	userConfig, _ := userLoader.Load() // Ignore error, use empty config

	// Create flow context
	flowContext := &core.FlowContext{
		Environment: cfg.Current,
		BaseURL:     cfg.GetCurrentEnv().BaseURL,
		Config:      cfg,
		UserConfig:  userConfig,
		Variables:   make(map[string]string),
	}

	// Create executor
	executor := core.NewFlowExecutor(agent, flowContext)

	// Create storage
	storage := NewFlowStorage()

	return &FlowService{
		agent:       agent,
		executor:    executor,
		storage:     storage,
		config:      cfg,
		userConfig:  userConfig,
		projectRoot: projectRoot,
	}, nil
}

// EstimateCost estimates the cost of AI analysis
func (s *FlowService) EstimateCost(ctx context.Context) (float64, int64, int, error) {
	// This would typically call the scanner to estimate costs
	// For now, return mock values or call a method on the agent
	// TODO: Implement actual cost estimation through the agent/scanner
	return 0.05, 75000, 43, nil // Mock values for demonstration
}

// DiscoverAndCache discovers flows and caches them
func (s *FlowService) DiscoverAndCache(ctx context.Context, useAI bool) (*core.FlowDiscoveryResult, error) {
	// Check cache first (always use cache if available for now)
	if len(s.storage.flows) > 0 && time.Since(s.storage.lastUpdated) < 5*time.Minute {
		flows := make([]core.Flow, 0, len(s.storage.flows))
		for _, flow := range s.storage.flows {
			flows = append(flows, *flow)
		}
		
		return &core.FlowDiscoveryResult{
			Flows:      flows,
			TotalFound: len(flows),
			Duration:   0, // From cache
			Confidence: s.calculateAverageConfidence(flows),
			Sources:    []string{"cache"},
		}, nil
	}

	startTime := time.Now()

	// Discover flows using agent
	flows, err := s.agent.DiscoverFlows(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	duration := time.Since(startTime)

	// Cache flows
	s.storage.flows = make(map[string]*core.Flow)
	for _, flow := range flows {
		flowCopy := flow // Create copy to avoid pointer issues
		s.storage.flows[flow.ID] = &flowCopy
	}
	s.storage.lastUpdated = time.Now()

	// Build result
	result := &core.FlowDiscoveryResult{
		Flows:      flows,
		TotalFound: len(flows),
		Duration:   duration,
		Confidence: s.calculateAverageConfidence(flows),
		Sources:    []string{s.projectRoot}, // Would include actual source files
	}

	return result, nil
}

// GetSignupFlow finds and returns the signup flow
func (s *FlowService) GetSignupFlow(ctx context.Context) (*core.Flow, error) {
	// Check cache first
	for _, flow := range s.storage.flows {
		if s.isSignupFlow(flow) {
			return flow, nil
		}
	}

	// Discover flows and find signup
	_, err := s.DiscoverAndCache(ctx, false)
	if err != nil {
		return nil, err
	}

	// Look for signup flow again
	for _, flow := range s.storage.flows {
		if s.isSignupFlow(flow) {
			return flow, nil
		}
	}

	// Use agent to specifically find signup flow
	return s.agent.FindSignupFlow(ctx)
}

// FindFlowByIntent finds a flow based on user intent
func (s *FlowService) FindFlowByIntent(ctx context.Context, intent string) (*core.Flow, error) {
	// Check cached flows first
	for _, flow := range s.storage.flows {
		if s.matchesIntent(flow, intent) {
			return flow, nil
		}
	}

	// Use agent to find by intent
	return s.agent.FindFlowByIntent(ctx, intent)
}

// ExecuteFlow executes a flow with the provided UI
func (s *FlowService) ExecuteFlow(ctx context.Context, flowID string, ui core.UIProvider) (*core.ExecutionResult, error) {
	flow := s.storage.GetFlow(flowID)
	if flow == nil {
		return nil, fmt.Errorf("flow not found: %s", flowID)
	}

	return s.executor.Execute(ctx, flow, ui)
}

// ExecuteFlowWithContext executes a flow with specific context
func (s *FlowService) ExecuteFlowWithContext(ctx context.Context, flowID string, ui core.UIProvider, flowContext *core.FlowContext) (*core.ExecutionResult, error) {
	flow := s.storage.GetFlow(flowID)
	if flow == nil {
		return nil, fmt.Errorf("flow not found: %s", flowID)
	}

	// Create executor with specific context
	executor := core.NewFlowExecutor(s.agent, flowContext)
	return executor.Execute(ctx, flow, ui)
}

// GetFlows returns all flows, optionally filtered by category
func (s *FlowService) GetFlows(ctx context.Context, category string) ([]core.Flow, error) {
	// Ensure we have flows
	if len(s.storage.flows) == 0 {
		_, err := s.DiscoverAndCache(ctx, true)
		if err != nil {
			return nil, err
		}
	}

	flows := make([]core.Flow, 0, len(s.storage.flows))
	for _, flow := range s.storage.flows {
		if category == "" || flow.Category == category {
			flows = append(flows, *flow)
		}
	}

	return flows, nil
}

// ExplainFlow gets an AI explanation of a flow
func (s *FlowService) ExplainFlow(ctx context.Context, flow *core.Flow) (string, error) {
	return s.agent.ExplainFlow(ctx, flow)
}

// GetFlowByID returns a specific flow by ID
func (s *FlowService) GetFlowByID(flowID string) (*core.Flow, error) {
	flow := s.storage.GetFlow(flowID)
	if flow == nil {
		return nil, fmt.Errorf("flow not found: %s", flowID)
	}
	return flow, nil
}

// Helper methods for FlowService

func (s *FlowService) isSignupFlow(flow *core.Flow) bool {
	if flow == nil {
		return false
	}
	
	flowName := flow.Name
	flowID := flow.ID
	category := flow.Category
	
	return contains(flowName, "signup") ||
		   contains(flowName, "register") ||
		   contains(flowName, "sign up") ||
		   contains(flowID, "signup") ||
		   contains(flowID, "register") ||
		   category == "authentication"
}

func (s *FlowService) matchesIntent(flow *core.Flow, intent string) bool {
	if flow == nil {
		return false
	}
	
	intentLower := toLower(intent)
	
	return contains(flow.Name, intentLower) ||
		   contains(flow.Description, intentLower) ||
		   contains(flow.ID, intentLower) ||
		   contains(flow.Category, intentLower)
}

func (s *FlowService) calculateAverageConfidence(flows []core.Flow) float64 {
	if len(flows) == 0 {
		return 0.0
	}
	
	total := 0.0
	for _, flow := range flows {
		total += flow.Confidence
	}
	
	return total / float64(len(flows))
}

// FlowStorage methods

// GetFlow returns a flow by ID
func (fs *FlowStorage) GetFlow(id string) *core.Flow {
	return fs.flows[id]
}

// StoreFlow stores a flow
func (fs *FlowStorage) StoreFlow(flow *core.Flow) {
	if flow != nil {
		fs.flows[flow.ID] = flow
		fs.lastUpdated = time.Now()
	}
}

// GetAllFlows returns all stored flows
func (fs *FlowStorage) GetAllFlows() map[string]*core.Flow {
	return fs.flows
}

// ClearFlows clears all stored flows
func (fs *FlowStorage) ClearFlows() {
	fs.flows = make(map[string]*core.Flow)
	fs.lastUpdated = time.Time{}
}

// GetCached returns cached flow if exists and not expired
func (fs *FlowStorage) GetCached(id string, maxAge time.Duration) *core.Flow {
	if time.Since(fs.lastUpdated) > maxAge {
		return nil
	}
	return fs.flows[id]
}

// Utility functions

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		   strings.Contains(toLower(s), toLower(substr))
}

func toLower(s string) string {
	return strings.ToLower(s)
}