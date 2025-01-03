// Package messages provides a flexible system for handling multi-format message content
// in AI agent communications. It implements a type-safe, extensible architecture for
// representing and manipulating messages that can contain text, images, audio, and
// specialized content types like refusal messages.
//
// Design decisions:
//   - Type safety: Generic message types ensure compile-time type checking
//   - Flexible content: Support for both simple string content and complex multi-part messages
//   - Extensible: Easy to add new content types by implementing ContentPart interface
//   - JSON interop: Full JSON serialization support with robust error handling
//   - Memory efficiency: Uses struct{} padding to enforce keyed initialization
//   - Clear separation: Different content types for user input vs assistant responses
//
// Key concepts:
//   - ContentOrParts: Represents user messages that can be either simple text or
//     multi-part content (text, images, audio)
//   - AssistantContentOrParts: Specialized content type for assistant responses,
//     supporting text and refusal messages
//   - ContentPart: Interface for implementing new content types
//   - AssistantContentPart: Interface for assistant-specific content types
//
// Example usage:
//
//	// Simple text message
//	msg := messages.ContentOrParts{Content: "Hello, world!"}
//
//	// Multi-part message with text and image
//	msg := messages.ContentOrParts{
//	    Parts: []messages.ContentPart{
//	        messages.Text("Check out this image:"),
//	        messages.Image("https://example.com/image.jpg"),
//	    },
//	}
//
//	// Assistant response with refusal
//	resp := messages.AssistantContentOrParts{
//	    Parts: []messages.AssistantContentPart{
//	        messages.Refusal("I cannot process that request"),
//	    },
//	}
//
// The package is designed to be used as part of a larger AI agent system,
// providing the foundational types needed for structured communication between
// users and AI agents. It handles serialization details and provides a clean API
// for working with different types of content.
package messages
