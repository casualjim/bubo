# Bubo

Bubo is a powerful Go library for building and orchestrating AI agents with a focus
on reliability, extensibility, and maintainable workflows.

## Overview

Bubo provides a robust foundation for creating AI-powered applications by offering:

- **Agent Orchestration**: Coordinate multiple AI agents working together
- **Event-Driven Architecture**: Built on a reliable event system for agent communication
- **Provider Abstraction**: Flexible integration with AI providers
- **Tool System**: Extensible framework for adding custom capabilities to agents
- **Memory Management**: Built-in short-term memory system for context retention
- **Workflow Engine**: Integration with Temporal for reliable workflow execution
- **Message Broker**: NATS integration for scalable message handling

## Architecture

```ascii
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     Agent       │     │    Provider     │     │      Tool       │
│  Orchestration  │◄────┤   Integration   │◄────┤    System       │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         ▲                      ▲                       ▲
         │                      │                       │
         ▼                      ▼                       ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     Event       │     │     Memory      │     │    Workflow     │
│     System      │◄────┤   Management    │◄────┤     Engine      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### Core Components

- **Agent**: Manages AI agent lifecycle and coordination
- **Provider**: Abstracts AI provider integration (e.g., OpenAI)
- **Tool**: Extensible system for adding capabilities to agents
- **Events**: Reliable event system for agent communication
- **Memory**: Short-term memory management for context retention
- **Workflow**: Temporal integration for reliable execution

## Installation

### Prerequisites

- Go 1.20 or later
- NATS Server (for message broker)
- Temporal (for workflow engine)

### Using Go Modules

```bash
go get github.com/casualjim/bubo
```

## Basic Usage

### Agent Handoff Example

This example demonstrates how to create agents that can transfer control between
each other based on language:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    // Ensure API Key is loaded
    _ "github.com/joho/godotenv/autoload"

    "github.com/casualjim/bubo"
    "github.com/casualjim/bubo/api"
    "github.com/casualjim/bubo/examples/internal/msgfmt"
    "github.com/casualjim/bubo/agent"
    "github.com/casualjim/bubo/provider/openai"
)

// Define agents with specific language capabilities
var (
    englishAgent = agent.New(
        agent.Name("English Agent"),
        agent.Model(openai.GPT4oMini()),
        agent.Instructions("You only speak English, so you only reply in english."),
        agent.Tools(transferToSpanishAgentTool),
    )
    spanishAgent = agent.New(
        agent.Name("Spanish Agent"),
        agent.Model(openai.GPT4oMini()),
        agent.Instructions("You only speak Spanish, so you only reply in spanish."),
    )
)

// Transfer spanish speaking users immediately
//
// bubo:agentTool
func transferToSpanishAgent() api.Agent { return spanishAgent }

func main() {
    ctx := context.Background()

    // Configure console output
    hook, result := msgfmt.Console[string](ctx, os.Stdout)

    // Create and run workflow
    p := bubo.New(
        bubo.Agents(englishAgent),
        bubo.Steps(
            bubo.Step(englishAgent.Name(), "Hola. ¿Como estás?"),
        ),
    )

    if err := p.Run(ctx, bubo.Local(hook)); err != nil {
        slog.Error("error running agent", "error", err)
        return
    }

    <-result
}
```

See the [examples](./examples) directory for more usage patterns including:

- Basic examples
  - Agent handoff
  - Context variables
  - Function calling
  - Structured output
- Temporal workflow integration
- Triage system implementation

## Component Relationships

### Agent ↔ Provider

Agents use providers to interact with AI models. The provider abstraction
allows for easy integration of different AI services.

### Agent ↔ Tool

Tools extend agent capabilities by providing specific functionalities.
Tools can be generated using the `bubo-tool-gen` command for marker
comments like:

```go
// bubo:agentTool
func transferToSpanishAgent() api.Agent { return spanishAgent }
```

### Agent ↔ Memory

The memory system helps agents maintain context across interactions and share
information.

### Provider ↔ Tool

Providers can utilize tools to enhance AI model capabilities and provide additional
functionalities.

## Contributing

### Development Setup

Clone the repository:

```bash
git clone https://github.com/casualjim/bubo.git
cd bubo
```

Install dependencies:

```bash
go mod download
```

Run tests:

```bash
go test ./...
```

### Guidelines

1. **Code Style**
   - Follow Go best practices and idioms
   - Use 2 spaces for indentation
   - Run `golangci-lint run` before submitting PRs

2. **Testing**
   - Write unit tests for new functionality
   - Ensure existing tests pass
   - For concurrent tests, use signals/events instead of timing

3. **Documentation**
   - Update documentation for new features
   - Include godoc examples for public APIs
   - Keep the README updated

4. **Pull Requests**
   - Create feature branches from `main`
   - Include tests and documentation
   - Ensure CI passes
   - Request review from maintainers

## License

This project is licensed under the [LICENSE](./LICENSE) file in the repository.
