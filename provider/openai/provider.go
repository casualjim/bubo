package openai

import (
	"context"
	"encoding/base64"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/jsonx"
	"github.com/casualjim/bubo/provider"
	"github.com/go-openapi/strfmt"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// Provider represents a service provider that interacts with the OpenAI API.
// It contains a client to communicate with the OpenAI service.
type Provider struct {
	client *openai.Client
}

// New creates a new instance of Provider with the given request options.
// It initializes an OpenAI client using the provided options and returns
// a pointer to the newly created Provider.
//
// Parameters:
//
//	options - A variadic list of request options to configure the OpenAI client.
//
// Returns:
//
//	*Provider - A pointer to the initialized Provider instance.
func New(options ...option.RequestOption) *Provider {
	client := openai.NewClient(options...)
	return &Provider{
		client: client,
	}
}

func (p *Provider) buildRequest(_ context.Context, params *provider.CompletionParams) (openai.ChatCompletionNewParams, error) {
	result, user := messagesToOpenAI(params.Instructions, params.Thread.MessagesIter())

	tools := make([]openai.ChatCompletionToolParam, len(params.Tools))
	for i, tool := range params.Tools {
		// Validate the tool function before conversion
		if tool.Function == nil {
			return openai.ChatCompletionNewParams{}, fmt.Errorf("tool %s has nil function", tool.Name)
		}

		name, parameters := tool.ToNameAndSchema()

		jv, err := jsonx.ToDynamicJSON(parameters)
		if err != nil {
			return openai.ChatCompletionNewParams{}, fmt.Errorf("failed to convert tool to name and schema: %w", err)
		}

		def := openai.FunctionDefinitionParam{
			Name:       openai.String(name),
			Parameters: openai.F(shared.FunctionParameters(jv)),
		}
		if strings.TrimSpace(tool.Description) != "" {
			def.Description = openai.String(tool.Description)
		}

		tools[i] = openai.ChatCompletionToolParam{
			Type:     openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(def),
		}
	}

	oaiParams := openai.ChatCompletionNewParams{
		Messages:    openai.F(result),
		Model:       openai.F(params.Model.Name()),
		N:           openai.Int(1),
		Temperature: openai.Float(0.1),
	}
	if len(tools) > 0 {
		oaiParams.Tools = openai.F(tools)
		oaiParams.ParallelToolCalls = openai.Bool(true)
	}
	if strings.TrimSpace(user) != "" {
		oaiParams.User = openai.String(user)
	}
	if params.ResponseSchema != nil {
		oaiParams.ResponseFormat = openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](openai.ResponseFormatJSONSchemaParam{
			Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
			JSONSchema: openai.F(openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        openai.String(params.ResponseSchema.Name),
				Description: openai.String(params.ResponseSchema.Description),
				Schema:      openai.F[any](params.ResponseSchema),
				Strict:      openai.Bool(true),
			}),
		})
		// oaiParams.
	}

	return oaiParams, nil
}

func (p *Provider) ChatCompletion(ctx context.Context, params provider.CompletionParams) (<-chan provider.StreamEvent, error) {
	chatParams, err := p.buildRequest(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	events := make(chan provider.StreamEvent, 10)
	go func() {
		defer close(events)
		if params.Stream {
			p.runStream(ctx, chatParams, &params, events)
		} else {
			p.runOnce(ctx, chatParams, &params, events)
		}
	}()
	return events, nil
}

func (p *Provider) runStream(ctx context.Context, params openai.ChatCompletionNewParams, command *provider.CompletionParams, events chan<- provider.StreamEvent) {
	strm := p.client.Chat.Completions.NewStreaming(ctx, params)

	if strm.Err() != nil {
		events <- provider.Error{
			Err:       strm.Err(),
			RunID:     command.RunID,
			TurnID:    command.Thread.ID(),
			Timestamp: strfmt.DateTime(time.Now()),
		}
		strm.Close()
		return
	}

	// Ensure cleanup on all exit paths
	defer func() {
		strm.Close()
		// Send error if context was cancelled
		if err := ctx.Err(); err != nil {
			events <- provider.Error{
				Err:       err,
				RunID:     command.RunID,
				TurnID:    command.Thread.ID(),
				Timestamp: strfmt.DateTime(time.Now()),
			}
		}
	}()

	var notFirst bool
	var acc openai.ChatCompletionAccumulator

	for strm.Next() {
		// Check context before processing each chunk
		if err := ctx.Err(); err != nil {
			return
		}

		if !notFirst {
			notFirst = true
			events <- provider.Delim{Delim: "start"}
		}

		chunk := strm.Current()
		if strm.Err() != nil {
			events <- provider.Error{
				Err:       strm.Err(),
				RunID:     command.RunID,
				TurnID:    command.Thread.ID(),
				Timestamp: strfmt.DateTime(time.Now()),
			}
			return
		}

		acc.AddChunk(chunk)
		events <- completionChunkToStreamEvent(&chunk, command)
	}

	// Only send completion events if we started streaming and context wasn't cancelled
	if notFirst && ctx.Err() == nil {
		events <- provider.Delim{Delim: "end"}
		compl := &acc.ChatCompletion
		events <- completionToStreamEvent(compl, command)
	}
}

func (p *Provider) runOnce(ctx context.Context, params openai.ChatCompletionNewParams, command *provider.CompletionParams, events chan<- provider.StreamEvent) {
	chat, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		events <- provider.Error{
			Err:       err,
			RunID:     command.RunID,
			TurnID:    command.Thread.ID(),
			Timestamp: strfmt.DateTime(time.Now()),
		}
		return
	}

	events <- completionToStreamEvent(chat, command)
}

func messagesToOpenAI(instructions string, iter iter.Seq[messages.Message[messages.ModelMessage]]) ([]openai.ChatCompletionMessageParamUnion, string) {
	result := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(instructions),
	}
	var user string
	for message := range iter {
		switch msg := message.Payload.(type) {
		case messages.ToolResponse:
			result = append(result, openai.ToolMessage(msg.ToolCallID, msg.Content))
		case messages.UserMessage:
			if message.Sender != "" {
				user = message.Sender
			}
			if msg.Content.Content != "" {
				um := openai.UserMessageParts(openai.TextPart(msg.Content.Content))
				result = append(result, um)
			}
			if len(msg.Content.Parts) > 0 {
				parts := make([]openai.ChatCompletionContentPartUnionParam, len(msg.Content.Parts))
				for i, part := range msg.Content.Parts {
					switch part := part.(type) {
					case messages.TextContentPart:
						parts[i] = openai.ChatCompletionContentPartTextParam{
							Text: openai.String(part.Text),
							Type: openai.F(openai.ChatCompletionContentPartTextTypeText),
						}
					case messages.ImageContentPart:
						parts[i] = openai.ChatCompletionContentPartImageParam{
							ImageURL: openai.F(openai.ChatCompletionContentPartImageImageURLParam{
								URL:    openai.String(part.URL),
								Detail: openai.F(openai.ChatCompletionContentPartImageImageURLDetail(part.Detail)),
							}),
							Type: openai.F(openai.ChatCompletionContentPartImageTypeImageURL),
						}
					case *messages.AudioContentPart:
						parts[i] = openai.ChatCompletionContentPartInputAudioParam{
							InputAudio: openai.F(openai.ChatCompletionContentPartInputAudioInputAudioParam{
								Data:   openai.String(base64.StdEncoding.EncodeToString(part.InputAudio.Data)),
								Format: openai.F(openai.ChatCompletionContentPartInputAudioInputAudioFormat(part.InputAudio.Format)),
							}),
							Type: openai.F(openai.ChatCompletionContentPartInputAudioTypeInputAudio),
						}
					}
				}
				result = append(result, openai.UserMessageParts(parts...))
			}
		case messages.ToolCallMessage:
			tcd := make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				tcd[i] = openai.ChatCompletionMessageToolCallParam{
					ID:   openai.String(tc.ID),
					Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction),
					Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      openai.String(tc.Name),
						Arguments: openai.String(tc.Arguments),
					}),
				}
			}
			result = append(result, openai.ChatCompletionMessageParam{
				Role:      openai.F(openai.ChatCompletionMessageParamRoleAssistant),
				ToolCalls: openai.F[any](tcd),
			})
		case messages.AssistantMessage:
			am := openai.ChatCompletionAssistantMessageParam{
				Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
			}

			// Handle refusal first since it takes precedence
			if msg.Refusal != "" {
				am.Refusal = openai.String(msg.Refusal)
			} else if msg.Content.Refusal != "" {
				am.Content = openai.F([]openai.ChatCompletionAssistantMessageParamContentUnion{
					openai.RefusalPart(msg.Content.Refusal),
				})
			} else if msg.Content.Content != "" {
				// Handle direct content if no refusal
				am.Content = openai.F([]openai.ChatCompletionAssistantMessageParamContentUnion{
					openai.TextPart(msg.Content.Content),
				})
			} else if len(msg.Content.Parts) > 0 {
				// Handle content parts if no direct content
				parts := make([]openai.ChatCompletionAssistantMessageParamContentUnion, 0, len(msg.Content.Parts))
				for _, part := range msg.Content.Parts {
					switch part := part.(type) {
					case messages.TextContentPart:
						parts = append(parts, openai.TextPart(part.Text))
					case messages.RefusalContentPart:
						parts = append(parts, openai.RefusalPart(part.Refusal))
					}
				}
				if len(parts) > 0 {
					am.Content = openai.F(parts)
				}
			}
			result = append(result, am)
		}
	}
	return result, user
}

func completionChunkToStreamEvent(chunk *openai.ChatCompletionChunk, command *provider.CompletionParams) provider.StreamEvent {
	if len(chunk.Choices) == 0 {
		return provider.Delim{Delim: "empty"}
	}

	choice := chunk.Choices[0].Delta
	if len(choice.ToolCalls) > 0 {
		tcd := make([]messages.ToolCallData, len(choice.ToolCalls))
		for i, tc := range choice.ToolCalls {
			tcd[i] = messages.ToolCallData{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}

		return provider.Chunk[messages.ToolCallMessage]{
			RunID:  command.RunID,
			TurnID: command.Thread.ID(),
			Chunk: messages.ToolCallMessage{
				ToolCalls: tcd,
			},
			Timestamp: strfmt.DateTime(time.Now()),
		}
	}

	return provider.Chunk[messages.AssistantMessage]{
		RunID:  command.RunID,
		TurnID: command.Thread.ID(),
		Chunk: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: choice.Content,
			},
		},
		Timestamp: strfmt.DateTime(time.Now()),
	}
}

func completionToStreamEvent(chat *openai.ChatCompletion, command *provider.CompletionParams) provider.StreamEvent {
	if len(chat.Choices) == 0 {
		return provider.Delim{Delim: "empty"}
	}

	choice := chat.Choices[0].Message
	if len(choice.ToolCalls) > 0 {
		tcd := make([]messages.ToolCallData, len(choice.ToolCalls))
		for i, tc := range choice.ToolCalls {
			tcd[i] = messages.ToolCallData{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}

		return provider.Response[messages.ToolCallMessage]{
			RunID:      command.RunID,
			TurnID:     command.Thread.ID(),
			Checkpoint: command.Thread.Checkpoint(),
			Response: messages.ToolCallMessage{
				ToolCalls: tcd,
			},
			Timestamp: strfmt.DateTime(time.Now()),
		}
	}

	return provider.Response[messages.AssistantMessage]{
		RunID:      command.RunID,
		TurnID:     command.Thread.ID(),
		Checkpoint: command.Thread.Checkpoint(),
		Response: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: choice.Content,
			},
		},
		Timestamp: strfmt.DateTime(time.Now()),
	}
}
