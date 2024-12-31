package bubo

import "github.com/casualjim/bubo/provider"

type Model interface {
	Name() string
	Provider() provider.Provider
}
