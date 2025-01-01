package bubo

import (
	"github.com/casualjim/bubo/api"
	"github.com/fogfish/opts"
)

type Parliament interface{}

type DefaultParliament struct {
	owls []api.Owl
}

func WithOwls(owl api.Owl, extraOwls ...api.Owl) opts.Option[DefaultParliament] {
	return opts.Type[DefaultParliament](func(o *DefaultParliament) error {
		o.owls = append(o.owls, owl)
		o.owls = append(o.owls, extraOwls...)
		return nil
	})
}

func NewParliament(options ...opts.Option[DefaultParliament]) Parliament {
	p := &DefaultParliament{}
	if err := opts.Apply(p, options); err != nil {
		panic(err)
	}
	return p
}
