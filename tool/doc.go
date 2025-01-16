/*
Package tool provides a framework for defining and managing tools that extend agent capabilities
in the Bubo system. Tools are functions that agents can invoke, with automatic parameter
validation and documentation generation through reflection.

# Design Decisions

  - Reflection-based: Uses Go's reflection to analyze function signatures
  - Schema Generation: Automatically generates JSON schemas for function parameters
  - Type Safety: Ensures type safety through compile-time checks
  - Functional Options: Provides a flexible configuration system
  - Documentation: Integrates documentation with function definitions

# Key Concepts

 1. Tool Definition
    A tool is defined by its function signature and metadata:
    - Name: Identifier for the tool
    - Description: Human-readable explanation
    - Parameters: Named input parameters
    - Function: The actual implementation

 2. Parameter Schema
    Tools automatically generate JSON schemas for their parameters:
    - Type validation
    - Required fields
    - Parameter descriptions
    - Custom validation rules

 3. Configuration Options
    Tools can be configured using functional options:
    - Name customization
    - Description setting
    - Parameter naming
    - Validation rules

# Usage Examples

Basic Tool Definition:

	// Define a tool function
	func calculateSum(x, y int) int {
		return x + y
	}

	// Create tool definition
	tool := Must(calculateSum,
		Name("calculateSum"),
		Description("Calculates the sum of two numbers"),
		Parameters("firstNumber", "secondNumber"),
	)

Tool with Context Variables:

	func processText(ctx types.ContextVars, text string) string {
		// Access context variables
		userID := ctx.Get("user_id")
		return fmt.Sprintf("Processing text for user %s: %s", userID, text)
	}

	tool := Must(processText,
		Name("processText"),
		Description("Processes text with user context"),
		Parameters("inputText"),
	)

Generated Tool:

	// bubo:agentTool
	func transferToAgent() api.Agent {
		return someAgent
	}

	// Generated code will create a tool definition automatically

# Best Practices

1. Function Design
  - Keep functions focused and single-purpose
  - Use meaningful parameter names
  - Include clear error handling
  - Document expected behavior

2. Tool Configuration
  - Provide descriptive names
  - Write clear descriptions
  - Use meaningful parameter names
  - Include validation where appropriate

3. Error Handling
  - Use Must() only when errors are not expected
  - Handle errors explicitly in production code
  - Validate input parameters
  - Return meaningful error messages

4. Documentation
  - Document tool purpose and usage
  - Explain parameter requirements
  - Provide usage examples
  - Note any side effects

# Integration

Tools integrate with several Bubo components:

 1. Agents
    Tools are provided to agents as capabilities:

    agent := agent.New(
    agent.Tools(tool1, tool2),
    // other configuration...
    )

 2. Providers
    Providers use tool definitions to enable function calling:

    provider.ChatCompletion(ctx, CompletionParams{
    Tools: []tool.Definition{tool1, tool2},
    // other parameters...
    })

 3. Code Generation
    The bubo-tool-gen command generates tool definitions from comments:

    // bubo:agentTool
    func myTool() {}

# Thread Safety

Tools should be designed with concurrency in mind:
  - Make tools stateless when possible
  - Use appropriate synchronization for stateful tools
  - Consider context cancellation
  - Handle concurrent invocations safely

For more details about specific components, see:
  - Definition: Core type representing a tool
  - Must: Helper for creating tools without error handling
  - New: Main constructor for creating tools
  - Options: Configuration options for customizing tools
*/
package tool
