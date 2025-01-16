package bubo

import (
	"fmt"

	"github.com/casualjim/bubo/messages"
)

type task interface {
	task()
}

type stringTask string

func (s stringTask) task() {}

type messageTask messages.Message[messages.UserMessage]

func (m messageTask) task() {}

type ConversationStep struct {
	agentName string
	task      task
}

type Task interface {
	~string | messages.Message[messages.UserMessage]
}

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
