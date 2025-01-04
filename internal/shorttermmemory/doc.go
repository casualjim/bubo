// Package shorttermmemory provides token usage tracking and aggregation for AI model
// interactions. It maintains detailed statistics about token consumption across
// different aspects of model usage, essential for monitoring, billing, and
// optimizing context window utilization.
//
// Design decisions:
//   - Granular tracking: Separate tracking for prompts and completions
//   - Detailed breakdowns: Sub-categories for different token types
//   - Aggregation support: Easy combining of usage across multiple operations
//   - JSON compatibility: Full serialization support for persistence/API integration
//   - Thread safety: Usage objects can be safely updated concurrently
//   - Extensible structure: Easy to add new token categories as models evolve
//
// Usage hierarchy:
//   - Usage: Top-level structure tracking overall token consumption
//     ├── CompletionTokensDetails: Breakdown of completion token usage
//     │   ├── AcceptedPredictionTokens: Tokens from successful predictions
//     │   ├── RejectedPredictionTokens: Tokens from unused predictions
//     │   ├── ReasoningTokens: Tokens used for model reasoning
//     │   └── AudioTokens: Tokens from audio processing
//     └── PromptTokensDetails: Breakdown of prompt token usage
//     ├── AudioTokens: Tokens from audio inputs
//     └── CachedTokens: Tokens retrieved from cache
//
// Example usage:
//
//	// Track usage for a model interaction
//	usage := &Usage{
//	    CompletionTokens: 150,
//	    PromptTokens: 100,
//	    TotalTokens: 250,
//	    CompletionTokensDetails: CompletionTokensDetails{
//	        ReasoningTokens: 50,
//	        AcceptedPredictionTokens: 100,
//	    },
//	    PromptTokensDetails: PromptTokensDetails{
//	        CachedTokens: 20,
//	    },
//	}
//
//	// Aggregate usage from multiple operations
//	totalUsage := &Usage{}
//	totalUsage.AddUsage(usage1)
//	totalUsage.AddUsage(usage2)
//
// The package is designed to be internal, providing essential token tracking
// functionality while keeping implementation details private. It's particularly
// useful for:
//   - Cost monitoring and billing
//   - Context window optimization
//   - Performance analysis
//   - Cache effectiveness measurement
//   - Model behavior analysis
package shorttermmemory
