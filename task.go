// Package bubo provides a framework for building conversational AI agents that can interact
// in a structured manner. It supports multi-agent conversations, structured output,
// and flexible execution contexts.
package bubo

import (
	"fmt"

	"github.com/casualjim/bubo/messages"
)

// task is an internal interface that marks valid task types that can be
// processed in conversation steps. It's implemented by stringTask and messageTask.
type task interface {
	task()
}

// stringTask represents a simple string-based task in a conversation.
// It implements the task interface.
type stringTask string

func (s stringTask) task() {}

// messageTask represents a structured message-based task in a conversation.
// It wraps a Message containing a UserMessage and implements the task interface.
type messageTask messages.Message[messages.UserMessage]

func (m messageTask) task() {}

// ConversationStep represents a single interaction step in a conversation workflow.
// It pairs an agent with a specific task to be executed.
type ConversationStep struct {
	agentName string // Name of the agent that should handle this step
	task      task   // The task to be executed by the agent
}

// Task is a type constraint interface that defines valid task types that can be
// used to create conversation steps. It allows either string literals or structured
// messages as valid task inputs.
type Task interface {
	~string | messages.Message[messages.UserMessage]
}

// Step creates a new ConversationStep with the specified agent and task.
// It accepts either a string or a Message[UserMessage] as the task input.
// The function will panic if an invalid task type is provided.
func Step[T Task](agentName string, tsk T) ConversationStep {
	var t task
	switch xt := any(tsk).(type) {
	case string:
		t = stringTask(xt)
	case messages.Message[messages.UserMessage]:
		t = messageTask(xt)
	default:
		panic(fmt.Sprintf("invalid task type: %T", xt))
	}
	return ConversationStep{
		agentName: agentName,
		task:      t,
	}
}
