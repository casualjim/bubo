# 🦉 Bubo: A Framework for Building AI Agents

Bubo is a Go framework for creating and orchestrating AI agents, with first-class
support for OpenAI's GPT models and function calling capabilities.

## 🌟 Overview

Bubo provides a robust foundation for building AI agents that can:

- Execute tools and functions with parallel execution support
- Handle streaming responses from LLM providers
- Process multi-modal inputs (text, images, audio)
- Manage complex conversation threads
- Maintain short-term memory for context
- Support event-driven architectures

## 🏗 Project Structure

- `api/` - Core API definitions and agent-related functionality
- `events/` - Event system for hooks and message handling
- `messages/` - Message types and content handling
- `provider/` - LLM provider integrations (OpenAI, etc.)``
- `tool/` - Tool system implementation
- `internal/`
  - `broker/` - Message broker implementation
  - `executor/` - Tool execution engine
  - `shorttermmemory/` - Context management and memory
- `examples/` - Example implementations and usage patterns

## ✨ Key Features

- 🛠 **Flexible Agent System**
  - Define custom agents with specific tools and capabilities
  - Configure model parameters and instructions
  - Support for parallel tool execution
  - Short-term memory for maintaining context
  
- 🔌 **Provider Integration**
  - First-class support for OpenAI's chat models
  - Extensible provider system for other LLMs
  - Streaming support for real-time responses
  - Function calling capabilities
  - Multi-modal content handling (text, images, audio)

- 📝 **Rich Message Handling**
  - Support for various message types (user, assistant, tool calls)
  - Structured content parts for different media types
  - Thread management for complex conversations
  - Event-driven message processing

- 🔧 **Tool System**
  - Define custom tools with JSON schema validation
  - Support for parallel tool execution
  - Structured tool responses and error handling
  - Tool generation utilities

- 📊 **Observability**
  - Event hooks for system monitoring
  - Stream events for real-time monitoring
  - Error tracking and handling

## 🚀 Getting Started

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
  slog.Info("running basic/agent-handoff example")
  ctx := context.Background()

  hook, result := msgfmt.Console[string](ctx, os.Stdout)

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

We include a code generation tool to convert the go function into an agent function:

```sh
go run github.com/casualjim/bubo/cmd/bubo-tool-gen@latest
```

## 📦 Installation

```bash
go get github.com/casualjim/bubo
```

### Code generation

```sh
go install github.com/casualjim/bubo/cmd/bubo-tool-gen@latest
```

## 🛠 Requirements

- Go 1.21 or higher
- OpenAI API key (for OpenAI provider)

## 📝 License

This project is licensed under the terms specified in the LICENSE file.
