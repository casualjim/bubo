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

- `api/` - Core API definitions and owl-related functionality
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
  "fmt"
  "log/slog"
  "os"

  "github.com/casualjim/bubo/events"
  "github.com/casualjim/bubo/examples/internal/msgfmt"
  pubsub "github.com/casualjim/bubo/internal/broker"
  "github.com/casualjim/bubo/internal/executor"
  "github.com/casualjim/bubo/internal/shorttermmemory"
  "github.com/casualjim/bubo/messages"
  "github.com/casualjim/bubo/owl"
  "github.com/casualjim/bubo/provider/openai"
  "github.com/joho/godotenv"
)

func main() {
  if err := godotenv.Load(); err != nil {
    slog.Warn("failed to load .env file")
  }
  ctx := context.Background()

  // Create an owl (agent)
  agent := owl.New(
    owl.Name("minimal-agent"), 
    owl.Model(openai.GPT4oMini()), 
    owl.Instructions("You are a helpful assistant"),
  )

  // Setup execution environment
  exec := executor.NewLocal(pubsub.Local[string]())
  memory := shorttermmemory.NewAggregator()
  stream, hook := events.NewChannelHook[string]()

  // Add initial message to memory
  memory.AddUserPrompt(messages.New().WithSender("user").UserPrompt("Hello, world!"))

  // Create and run command
  cmd, err := executor.NewRunCommand(agent, memory, hook)
  if err != nil {
    slog.Error("failed to create command", "error", err)
  }

  if err := exec.Run(ctx, cmd); err != nil {
    slog.Error("failed to run command", "error", err)
  }

  // Format and display output
  if err := msgfmt.ConsolePretty(ctx, os.Stdout, stream); err != nil {
    slog.Error("failed to format output", "error", err)
  }
}
```

## 📦 Installation

```bash
go get github.com/casualjim/bubo
```

## 🛠 Requirements

- Go 1.21 or higher
- OpenAI API key (for OpenAI provider)

## 📝 License

This project is licensed under the terms specified in the LICENSE file.
