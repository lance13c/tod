# LLM Cost Tracking Implementation

## Overview

Tod now includes comprehensive LLM cost tracking and usage monitoring to help developers understand and control their AI analysis costs.

## Features

### Real-time Cost Display
- **Pre-analysis cost estimation** - Shows estimated cost before making LLM calls
- **User confirmation for expensive operations** - Prompts for confirmation on costs >$0.05
- **Actual cost tracking** - Displays real token usage and costs after analysis
- **Running session totals** - Tracks cumulative costs per session

### Usage Analytics
- **Session tracking** - Current session costs and token usage
- **Historical data** - Daily, weekly, and monthly breakdowns
- **Provider breakdown** - Usage statistics per LLM provider (OpenAI, Anthropic, OpenRouter)
- **Model-specific tracking** - Token and cost tracking per AI model

### Cost Management
- **Current 2025 pricing** - Up-to-date pricing for major LLM providers:
  - GPT-4o: $3/$10 per 1M input/output tokens
  - GPT-4o-mini: $0.15/$0.60 per 1M tokens  
  - Claude 3.5 Sonnet: $3/$15 per 1M tokens
  - Claude 3 Haiku: $0.25/$1.25 per 1M tokens
- **Automatic cost calculation** - Real-time cost computation based on actual token usage
- **Export capabilities** - Export usage data to JSON/CSV for external analysis

## Usage

### View Current Session
```bash
tod usage
```

### View Historical Usage
```bash
tod usage --daily     # Daily breakdown
tod usage --weekly    # Weekly summary  
tod usage --monthly   # Monthly totals
```

### Export Data
```bash
tod usage --export json  # Export to JSON
tod usage --export csv   # Export to CSV
```

### Reset Usage Data
```bash
tod usage --reset
```

## Configuration

Set your API key for cost-effective analysis:
```bash
export OPENROUTER_API_KEY="your_key_here"
```

Tod defaults to using GPT-4o-mini for code analysis to minimize costs while maintaining quality.

## Cost Control

### Automatic Safeguards
- **Small operations** (<$0.05) - Proceed automatically
- **Larger operations** (≥$0.05) - Require user confirmation
- **Fallback analysis** - Uses pattern-matching when LLM is unavailable/declined

### Example Output
```
🤖 Analyzing HomePage.tsx...
💰 Estimated cost: $0.003 (1.2K tokens)
📊 Actual cost: $0.002 (956 tokens)
```

### Session Summary
```
┌─ LLM Usage - Current Session ─┐
│ Started: 2025-01-06 14:30:15   │
│ Duration: 15m 32s              │
└────────────────────────────────┘

Total Requests: 8
Total Tokens:   12.3K (8.1K input, 4.2K output)
Total Cost:     $0.045
```

## Implementation Details

### Architecture
- **Cost Calculator** (`internal/llm/costs.go`) - Handles pricing and cost computation
- **Token Counter** (`internal/llm/tokens.go`) - Estimates and tracks token usage
- **Usage Tracking** (`cmd/usage.go`) - CLI commands and data persistence
- **Enhanced LLM Clients** - All providers now include usage tracking

### Data Storage
Usage data is stored locally in `~/.tod/usage.json` with:
- Session data (current session)
- Daily aggregates (by date)
- Weekly summaries (by ISO week)
- Monthly totals (by year-month)
- Per-provider breakdowns

### Provider Integration
- **OpenRouter** - Full cost tracking with real API responses
- **OpenAI/Anthropic** - Cost tracking via fallback to mock clients  
- **Local/Mock** - Zero-cost analysis for development

## Benefits

1. **Cost Transparency** - Know exactly what each analysis costs
2. **Budget Control** - Set expectations and avoid surprises
3. **Usage Optimization** - Identify high-cost operations to optimize
4. **Provider Comparison** - Compare costs across different LLM providers
5. **Historical Analysis** - Track usage trends over time

## Future Enhancements

- Budget limits and alerts
- Cost optimization recommendations
- Integration with cloud provider billing APIs
- Team usage aggregation and reporting
- Custom pricing for enterprise models