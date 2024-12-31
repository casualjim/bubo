# 🦉 Bubo: A Framework for Building AI Agents

Bubo is a Go framework for creating and orchestrating AI agents, with first-class
support for OpenAI's GPT models and function calling capabilities.

## 🌟 Overview

Bubo provides a robust foundation for building AI agents that can:

- Execute tools and functions
- Handle streaming responses
- Process multi-modal inputs (text, images, audio)
- Manage complex conversation threads
- Run parallel tool executions

## ✨ Key Features

- 🛠 **Flexible Agent System**
  - Define custom agents with specific tools and capabilities
  - Configure model parameters and instructions
  - Support for parallel tool execution
  
- 🔌 **OpenAI Integration**
  - First-class support for OpenAI's chat models
  - Streaming support for real-time responses
  - Function calling capabilities
  - Multi-modal content handling (text, images, audio)

- 📝 **Rich Message Handling**
  - Support for various message types (user, assistant, tool calls)
  - Structured content parts for different media types
  - Thread management for complex conversations

- 🔧 **Tool System**
  - Define custom tools with JSON schema validation
  - Support for parallel tool execution
  - Structured tool responses and error handling

- 📊 **Observability**
  - Detailed logging with zerolog integration
  - Stream events for real-time monitoring
  - Error tracking and handling

## 🚀 Getting Started

```go
package main

import (
  "github.com/casualjim/bubo"
  "github.com/openai/openai-go"
)

func main() {
  // Create a new agent
  agent := bubo.NewAgent(
    "my-agent",
    string(openai.ChatModelGPT4),
    "Your agent instructions here",
  )

  // Add tools
  agent.AddTool(bubo.AgentToolDefinition{
    Name: "my-tool",
    Function: &bubo.FunctionDefinition{
      // Define your tool's schema and behavior
    },
  })

  // Configure parallel tool execution
  agent.EnableParallelToolCalls()
}
```

## 📦 Installation

```bash
go get github.com/casualjim/bubo
```

## 🛠 Requirements

- Go 1.23.3 or higher
- OpenAI API key for AI capabilities

## 📝 License

This project is licensed under the terms specified in the LICENSE file.
