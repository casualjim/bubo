// Package provider implements an abstraction layer for interacting with AI model providers
// (like OpenAI, Anthropic, etc.) in a consistent way. It defines interfaces and types
// for streaming AI completions while handling provider-specific implementation details.
//
// Design decisions:
//   - Provider abstraction: Single interface that different AI providers can implement
//   - Streaming first: Built around streaming responses for real-time interaction
//   - Type-safe events: Generic types ensure compile-time correctness of response handling
//   - Structured metadata: Each event includes run/turn IDs and timestamps for tracking
//   - Error handling: Dedicated error type that preserves context and metadata
//   - Memory management: Integration with short-term memory for context preservation
//
// Key concepts:
//   - Provider: Interface defining the contract for AI model providers
//   - StreamEvent: Base interface for all streaming events (chunks, responses, errors)
//   - CompletionParams: Configuration for chat completion requests
//   - Checkpoint: Captures conversation state for context management
//
// The streaming architecture uses four main event types:
//  1. Delim: Delimiter events marking stream boundaries
//  2. Chunk: Incremental response fragments
//  3. Response: Complete responses with checkpoints
//  4. Error: Error events with preserved context
//
// Example usage:
//
//	provider := openai.NewProvider(config)
//	params := CompletionParams{
//	    RunID:        uuid.New(),
//	    Instructions: "You are a helpful assistant",
//	    Stream:       true,
//	    Tools:        []tool.Definition{...},
//	}
//
//	events, err := provider.ChatCompletion(ctx, params)
//	if err != nil {
//	    return err
//	}
//
//	for event := range events {
//	    switch e := event.(type) {
//	    case Chunk[messages.AssistantMessage]:
//	        // Handle incremental response
//	    case Response[messages.AssistantMessage]:
//	        // Handle complete response
//	    case Error:
//	        // Handle error with context
//	    }
//	}
//
// The package is designed to be extensible, allowing new providers to be added
// by implementing the Provider interface while maintaining consistent behavior
// and error handling across different AI model providers.
package provider
