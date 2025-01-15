package openai

import (
	"sync"

	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/provider/models"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func GPT4oMini(opts ...option.RequestOption) api.Model {
	return Model(openai.ChatModelGPT4oMini, opts...)
}

func GPT4o(opts ...option.RequestOption) api.Model {
	return Model(openai.ChatModelChatgpt4oLatest, opts...)
}

func O1Mini(opts ...option.RequestOption) api.Model {
	return Model(openai.ChatModelO1Mini, opts...)
}

func O1(opts ...option.RequestOption) api.Model {
	return Model(openai.ChatModelO1, opts...)
}

func Model(name string, opts ...option.RequestOption) api.Model {
	return models.GetOrAdd(name, func() api.Model {
		return &model{
			name: name,
			opts: opts,
		}
	})
}

var _ api.Model = (*model)(nil)

type model struct {
	name string
	opts []option.RequestOption

	prov     provider.Provider
	provOnce sync.Once
}

func (m *model) Name() string {
	return m.name
}

func (m *model) Provider() provider.Provider {
	m.provOnce.Do(func() {
		m.prov = New(m.opts...)
	})
	return m.prov
}
