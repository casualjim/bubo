package openai

import (
	"sync"

	"github.com/alphadose/haxmap"
	"github.com/casualjim/bubo"
	"github.com/casualjim/bubo/provider"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var modelRegistry = haxmap.New[string, bubo.Model]()

func GPT4oMini(opts ...option.RequestOption) bubo.Model {
	return Model(openai.ChatModelGPT4oMini, opts...)
}

func GPT4o(opts ...option.RequestOption) bubo.Model {
	return Model(openai.ChatModelChatgpt4oLatest, opts...)
}

func O1Mini(opts ...option.RequestOption) bubo.Model {
	return Model(openai.ChatModelO1Mini, opts...)
}

func O1(opts ...option.RequestOption) bubo.Model {
	return Model(openai.ChatModelO1, opts...)
}

func Model(name string, opts ...option.RequestOption) bubo.Model {
	m, _ := modelRegistry.GetOrCompute(name, func() bubo.Model {
		return &model{
			name: name,
			opts: opts,
		}
	})
	return m
}

var _ bubo.Model = (*model)(nil)

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