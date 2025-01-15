package agent

import (
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/internal/registry"
)

var Global = registry.New[api.Agent]()

func Add(agent api.Agent) {
	Global.Add(agent.Name(), agent)
}

func Get(name string) (api.Agent, bool) {
	return Global.Get(name)
}

func Del(name string) {
	Global.Del(name)
}
