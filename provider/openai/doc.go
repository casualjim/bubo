/*
Package openai implements the provider.Provider interface for OpenAI's chat models.
It handles communication with OpenAI's API, including streaming responses, tool calls,
and structured output formatting.

# Design Decisions

  - Streaming First: Built around efficient streaming of responses
  - Type Safety: Strong typing for all OpenAI-specific types
  - Memory Efficient: Processes responses in chunks to minimize memory usage
  - Thread Safe: Safe for concurrent use across goroutines
  - Lazy Initialization: Models initialize their provider on first use

# Available Models

The package provides several pre-configured models:

  - GPT4oMini(): Smaller, faster GPT-4 model
  - GPT4o(): Full GPT-4 model with latest capabilities
  - O1Mini(): Smaller version of the O1 model
  - O1(): Full O1 model

Custom models can be created using the Model() function:

	model := openai.Model("custom-model-name",
		option.WithAPIKey("your-key"),
		option.WithOrganization("your-org"),
	)

# Message Handling

The package supports various message types:

 1. System Instructions
    Initial instructions that set the context:

    params := provider.CompletionParams{
    Instructions: "You are a helpful assistant",
    Model:       openai.GPT4o(),
    }

 2. User Messages
    Supports text, images, and audio content:

    message := messages.UserMessage{
    Content: messages.UserContentOrParts{
    Content: "Hello",
    Parts: []messages.ContentPart{
    messages.TextContentPart{Text: "With image"},
    messages.ImageContentPart{URL: "image.jpg"},
    },
    },
    }

 3. Assistant Messages
    Handles both text responses and tool calls:

    response := messages.AssistantMessage{
    Content: messages.AssistantContentOrParts{
    Content: "Response text",
    },
    }

# Streaming Implementation

The package implements efficient streaming:

1. Event Types
  - Delim: Stream boundary markers
  - Chunk: Incremental response pieces
  - Response: Complete messages
  - Error: Error events with context

2. Stream Processing:

	events, err := provider.ChatCompletion(ctx, params)
	if err != nil {
		return err
	}

	for event := range events {
		switch e := event.(type) {
		case provider.Chunk[messages.AssistantMessage]:
			// Handle incremental response
		case provider.Response[messages.AssistantMessage]:
			// Handle complete response
		case provider.Error:
			// Handle error
		}
	}

# Tool Integration

The package supports OpenAI's function calling feature:

 1. Tool Definition
    Tools are automatically converted to OpenAI function definitions:

    tool := tool.Must(myFunction,
    tool.Name("myTool"),
    tool.Description("Tool description"),
    )

 2. Function Calling
    Tools can be included in completion requests:

    params := provider.CompletionParams{
    Tools: []tool.Definition{tool},
    Model: openai.GPT4o(),
    }

# Best Practices

1. Model Selection
  - Use GPT4oMini for faster, cost-effective processing
  - Use GPT4o for complex tasks requiring latest capabilities
  - Consider O1 models for specialized use cases

2. Error Handling
  - Always check initial error from ChatCompletion
  - Handle stream errors in event loop
  - Implement proper context cancellation

3. Resource Management
  - Close unused streams
  - Implement proper cleanup in error cases
  - Monitor API usage and rate limits

4. Performance
  - Use streaming for real-time responses
  - Process chunks efficiently
  - Implement proper buffering

# Thread Safety

The package is designed to be thread-safe:
  - Providers can be shared across goroutines
  - Models use sync.Once for safe initialization
  - Stream operations are goroutine-safe

# Configuration

The package supports various configuration options:

1. API Configuration
  - API keys
  - Organization ID
  - Base URL
  - Timeouts

2. Request Options
  - Temperature
  - Maximum tokens
  - Stop sequences
  - Presence penalties

Example configuration:

	provider := openai.New(
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		option.WithOrganization("org-id"),
		option.WithTimeout(30*time.Second),
	)

For more details about specific components, see:
  - Provider: Main interface implementation
  - Model: Model-specific implementations
  - StreamEvent: Event types for streaming
  - CompletionParams: Request configuration
*/
package openai
