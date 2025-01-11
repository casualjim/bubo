package models

import (
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/internal/registry"
)

var Global = registry.New[api.Model]()

func Add(model api.Model) {
	Global.Add(model.Name(), model)
}

func Get(name string) (api.Model, bool) {
	return Global.Get(name)
}

func GetOrAdd(name string, modelF func() api.Model) api.Model {
	m, _ := Global.GetOrAdd(name, modelF)
	return m
}

func Del(name string) {
	Global.Del(name)
}
