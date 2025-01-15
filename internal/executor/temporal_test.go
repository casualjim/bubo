package executor

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	buboagent "github.com/casualjim/bubo/agent"
	"github.com/casualjim/bubo/api"
	"github.com/casualjim/bubo/events"
	"github.com/casualjim/bubo/internal/mocks"
	"github.com/casualjim/bubo/internal/shorttermmemory"
	"github.com/casualjim/bubo/messages"
	"github.com/casualjim/bubo/pkg/uuidx"
	"github.com/casualjim/bubo/provider"
	"github.com/casualjim/bubo/provider/models"
	"github.com/casualjim/bubo/tool"
	"github.com/casualjim/bubo/types"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type testEnv struct {
	env      *testsuite.TestWorkflowEnvironment
	broker   *mocks.Broker
	temporal *Temporal
}

func setupTestEnvironment(t *testing.T) *testEnv {
	// Setup temporal test environment
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.SetTestTimeout(time.Minute * 5)
	env.SetWorkflowRunTimeout(time.Minute * 5)

	// Setup mock broker
	mockBroker := mocks.NewBroker(t)
	temporal := &Temporal{broker: mockBroker}

	return &testEnv{
		env:      env,
		broker:   mockBroker,
		temporal: temporal,
	}
}

func TestTemporalRunCompletion(t *testing.T) {
	t.Run("streaming responses", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		// Configure workflow options to disable retries
		env.env.SetStartWorkflowOptions(client.StartWorkflowOptions{
			TaskQueue: "test-queue",
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 1,
			},
		})

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		// Register mock agent in the global registry
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Model().Return(model)
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		runID := uuidx.New()
		mem := shorttermmemory.New()
		expectedResult := "streaming chunk"

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Setup provider expectations
		eventChan := make(chan provider.StreamEvent, 2)
		turnID := uuidx.New()
		// First send a chunk
		eventChan <- provider.Chunk[messages.AssistantMessage]{
			RunID:  runID,
			TurnID: turnID,
			Chunk: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: expectedResult[:5],
				},
			},
		}
		// Then send the final response that will be added to the checkpoint
		eventChan <- provider.Response[messages.AssistantMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: expectedResult,
				},
			},
			Checkpoint: mem.Checkpoint(),
		}
		close(eventChan)
		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID && p.Stream == true
		})).Return(eventChan, nil)

		// Setup topic expectations for publishing both chunk and response
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(2)

		// // First expect the chunk publish
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			chunk, ok := evt.(events.Chunk[messages.AssistantMessage])
			return ok &&
				chunk.RunID == runID &&
				chunk.Sender == "test_agent" &&
				chunk.Chunk.Content.Content == expectedResult[:5]
		})).Return(nil)

		// // Then expect the response publish
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Response[messages.AssistantMessage])
			return ok &&
				resp.RunID == runID &&
				resp.Sender == "test_agent" &&
				resp.Response.Content.Content == expectedResult
		})).Return(nil)

		params := completionParams{
			RunID: runID,
			Agent: RemoteAgent{
				Name:  agent.Name(),
				Model: agent.Model().Name(),
			},
		}
		// Execute workflow
		var result string
		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID:         params.RunID,
			Agent:      params.Agent,
			Stream:     true,
			MaxTurns:   10,
			Checkpoint: mem.Checkpoint(),
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		require.NoError(t, env.env.GetWorkflowResult(&result))
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error handling", func(t *testing.T) {
		env := setupTestEnvironment(t)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		expectedError := fmt.Errorf("test error")

		// Setup mock topic
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic)
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			return true // Error messages are handled differently
		})).Return(nil)

		// Register mock agent in the global registry
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Model().Return(model)
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Setup provider expectations
		eventChan := make(chan provider.StreamEvent, 3)
		tries := 0
		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(eventChan, nil).Run(func(ctx context.Context, p provider.CompletionParams) {
			eventChan <- provider.Error{
				RunID: runID,
				Err:   expectedError,
			}
			tries++
			if tries >= 3 {
				close(eventChan)
			}
		})

		mem := shorttermmemory.New()
		params := completionParams{
			RunID: runID,
			Agent: RemoteAgent{
				Name:  agent.Name(),
				Model: agent.Model().Name(),
			},
		}
		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID:         params.RunID,
			Agent:      params.Agent,
			MaxTurns:   10,
			Checkpoint: mem.Checkpoint(),
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.Error(t, env.env.GetWorkflowError())
	})

	t.Run("completion with context", func(t *testing.T) {
		env := setupTestEnvironment(t)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		expectedResult := "test response with context"
		expectedCheckpoint := shorttermmemory.New().Checkpoint()
		contextVars := types.ContextVars{
			"test_key": "test_value",
		}

		// Setup mock topic
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic)
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			resp, ok := msg.(events.Response[messages.AssistantMessage])
			return ok &&
				resp.RunID == runID &&
				resp.Sender == "test_agent" &&
				resp.Response.Content.Content == expectedResult
		})).Return(nil)

		// Register mock agent in the global registry
		agent.EXPECT().Name().Return("test_agent")
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Setup provider expectations
		eventChan := make(chan provider.StreamEvent, 1)
		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(eventChan, nil).Run(func(ctx context.Context, p provider.CompletionParams) {
			eventChan <- provider.Response[messages.AssistantMessage]{
				RunID: runID,
				Response: messages.AssistantMessage{
					Content: messages.AssistantContentOrParts{
						Content: expectedResult,
					},
				},
				Checkpoint: expectedCheckpoint,
			}
			close(eventChan)
		})

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		var result string
		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:              "test_agent",
				Model:             "test_model",
				Instructions:      "test instructions with {{.test_key}}",
				ParallelToolCalls: false,
			},
			ContextVariables: contextVars,
			MaxTurns:         10,
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		require.NoError(t, env.env.GetWorkflowResult(&result))
		assert.Equal(t, expectedResult, result)
	})
}

func TestTemporalToolCallsWithComplexTypes(t *testing.T) {
	t.Run("struct return", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		turnID := uuidx.New()

		// Register mock agent in the global registry
		agent.EXPECT().Name().Return("test_agent").Times(1) // Called for registration and tool call
		agent.EXPECT().Tools().Return([]tool.Definition{    // Called for tool lookup and tool call
			{
				Name: "complex_tool",
				Function: func() struct {
					Name   string
					Value  int
					Nested struct{ Flag bool }
				} {
					return struct {
						Name   string
						Value  int
						Nested struct{ Flag bool }
					}{
						Name:  "test",
						Value: 42,
						Nested: struct{ Flag bool }{
							Flag: true,
						},
					}
				},
			},
		}).Times(1)
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model").Times(1) // Called for registration
		model.EXPECT().Provider().Return(prov).Times(2)     // Called for registration
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Set up memory with initial content
		mem := shorttermmemory.New()
		userMsg := messages.Message[messages.UserMessage]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.UserMessage{
				Content: messages.ContentOrParts{
					Content: "Test complex tool returns",
				},
			},
			Sender:    "user",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddUserPrompt(userMsg)

		// First completion returns tool call
		toolCallEvents := make(chan provider.StreamEvent, 1)
		toolCallEvents <- provider.Response[messages.ToolCallMessage]{
			RunID:      runID,
			TurnID:     turnID,
			Checkpoint: mem.Checkpoint(),
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "tool1",
						Name:      "complex_tool",
						Arguments: "{}",
					},
				},
			},
		}
		close(toolCallEvents)

		// Final completion returns response
		finalEvents := make(chan provider.StreamEvent, 1)
		finalEvents <- provider.Response[messages.AssistantMessage]{
			RunID:      runID,
			TurnID:     turnID,
			Checkpoint: mem.Checkpoint(),
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "final response",
				},
			},
		}
		close(finalEvents)

		// Setup provider expectations
		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(toolCallEvents, nil).Once()

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(finalEvents, nil).Once()

		// Setup topic expectations
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(3)

		// Expect tool call response, tool response, and final response
		var toolResponse string
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			if resp, ok := evt.(events.Request[messages.ToolResponse]); ok {
				toolResponse = resp.Message.Content
				return true
			}
			return true
		})).Return(nil).Times(3)

		var result string
		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:  "test_agent",
				Model: "test_model",
			},
			MaxTurns:   10,
			Checkpoint: mem.Checkpoint(),
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		require.NoError(t, env.env.GetWorkflowResult(&result))
		assert.Equal(t, "final response", result)
		assert.Contains(t, toolResponse, `{"Name":"test","Value":42,"Nested":{"Flag":true}}`)
	})

	t.Run("text marshaler success", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		turnID := uuidx.New()

		// Register mock agent with text marshaler tool
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Tools().Return([]tool.Definition{
			{
				Name: "marshaler_tool",
				Function: func() textMarshaler {
					return textMarshaler{shouldError: false}
				},
			},
		})
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Setup provider expectations
		toolCallEvents := make(chan provider.StreamEvent, 1)
		toolCallEvents <- provider.Response[messages.ToolCallMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "tool1",
						Name:      "marshaler_tool",
						Arguments: "{}",
					},
				},
			},
		}
		close(toolCallEvents)

		finalEvents := make(chan provider.StreamEvent, 1)
		finalEvents <- provider.Response[messages.AssistantMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "final response",
				},
			},
		}
		close(finalEvents)

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(toolCallEvents, nil).Once()

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(finalEvents, nil).Once()

		// Setup topic expectations
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(3)

		var toolResponse string
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			if resp, ok := evt.(events.Request[messages.ToolResponse]); ok {
				toolResponse = resp.Message.Content
			}
			return true
		})).Return(nil).Times(3)

		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:  "test_agent",
				Model: "test_model",
			},
			MaxTurns: 10,
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		assert.Equal(t, "marshaled text", toolResponse)
	})

	t.Run("text marshaler error", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		turnID := uuidx.New()

		// Register mock agent with failing text marshaler tool
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Tools().Return([]tool.Definition{
			{
				Name: "marshaler_tool",
				Function: func() textMarshaler {
					return textMarshaler{shouldError: true}
				},
			},
		})
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Setup provider expectations
		toolCallEvents := make(chan provider.StreamEvent, 1)
		toolCallEvents <- provider.Response[messages.ToolCallMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "tool1",
						Name:      "marshaler_tool",
						Arguments: "{}",
					},
				},
			},
		}
		close(toolCallEvents)

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(toolCallEvents, nil)

		// Setup topic expectations
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic)

		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Response[messages.ToolCallMessage])
			return ok && resp.TurnID == turnID
		})).Return(nil)

		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:  "test_agent",
				Model: "test_model",
			},
			MaxTurns: 10,
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.Error(t, env.env.GetWorkflowError())
		assert.Contains(t, env.env.GetWorkflowError().Error(), "intentional marshal error")
	})
}

func TestTemporalToolCallsWithContextPropagation(t *testing.T) {
	t.Run("basic propagation", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		turnID := uuidx.New()

		// Register mock agent in the global registry
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Tools().Return([]tool.Definition{
			{
				Name: "context_tool1",
				Function: func() types.ContextVars {
					return types.ContextVars{
						"key1": "value1",
						"key2": "value2",
					}
				},
			},
			{
				Name: "context_tool2",
				Parameters: map[string]string{
					"param0": "cv",
				},
				Function: func(cv types.ContextVars) string {
					return fmt.Sprintf("got values: %v, %v", cv["key1"], cv["key2"])
				},
			},
		})
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Setup provider expectations for tool calls
		toolCallEvents := make(chan provider.StreamEvent, 1)
		toolCallEvents <- provider.Response[messages.ToolCallMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "tool1",
						Name:      "context_tool1",
						Arguments: "{}",
					},
					{
						ID:        "tool2",
						Name:      "context_tool2",
						Arguments: `{"cv": {}}`,
					},
				},
			},
		}
		close(toolCallEvents)

		// Setup provider expectations for final response
		finalEvents := make(chan provider.StreamEvent, 1)
		finalEvents <- provider.Response[messages.AssistantMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "final response",
				},
			},
		}
		close(finalEvents)

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(toolCallEvents, nil).Once()

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(finalEvents, nil).Once()

		// Setup topic expectations
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(4)

		var toolResponses []string
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			if resp, ok := evt.(events.Request[messages.ToolResponse]); ok {
				toolResponses = append(toolResponses, resp.Message.Content)
			}
			return true
		})).Return(nil).Times(4)

		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:  "test_agent",
				Model: "test_model",
			},
			MaxTurns: 10,
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		assert.Contains(t, toolResponses[1], "got values: value1, value2", "Context variables should propagate between tool calls")
	})

	t.Run("chained updates", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		turnID := uuidx.New()

		// Register mock agent with tools that modify context
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Tools().Return([]tool.Definition{
			{
				Name: "init_context",
				Function: func() types.ContextVars {
					return types.ContextVars{"counter": float64(1)}
				},
			},
			{
				Name: "increment_counter",
				Parameters: map[string]string{
					"param0": "cv",
				},
				Function: func(cv types.ContextVars) types.ContextVars {
					if cv == nil {
						cv = types.ContextVars{}
					}
					counter := cv["counter"].(float64)
					cv["counter"] = counter + 1
					return cv
				},
			},
			{
				Name: "get_counter",
				Parameters: map[string]string{
					"param0": "cv",
				},
				Function: func(cv types.ContextVars) string {
					return fmt.Sprintf("counter value: %v", cv["counter"])
				},
			},
		})
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Setup provider expectations for tool calls
		toolCallEvents := make(chan provider.StreamEvent, 1)
		toolCallEvents <- provider.Response[messages.ToolCallMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "tool1",
						Name:      "init_context",
						Arguments: "{}",
					},
					{
						ID:        "tool2",
						Name:      "increment_counter",
						Arguments: `{"cv": {}}`,
					},
					{
						ID:        "tool3",
						Name:      "get_counter",
						Arguments: `{"cv": {}}`,
					},
				},
			},
		}
		close(toolCallEvents)

		// Setup provider expectations for final response
		finalEvents := make(chan provider.StreamEvent, 1)
		finalEvents <- provider.Response[messages.AssistantMessage]{
			RunID:  runID,
			TurnID: turnID,
			Response: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "final response",
				},
			},
		}
		close(finalEvents)

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(toolCallEvents, nil).Once()

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(finalEvents, nil).Once()

		// Setup topic expectations
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(5)

		var toolResponses []string
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			if resp, ok := evt.(events.Request[messages.ToolResponse]); ok {
				toolResponses = append(toolResponses, resp.Message.Content)
			}
			return true
		})).Return(nil).Times(5)

		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:  "test_agent",
				Model: "test_model",
			},
			MaxTurns: 10,
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		assert.Contains(t, toolResponses[2], "counter value: 2", "Context variables should update through chain of tools")
	})
}

func TestTemporalToolCalls(t *testing.T) {
	t.Run("sequential tool calls", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		turnID := uuidx.New()

		// Register mock agent in the global registry
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Tools().Return([]tool.Definition{
			{
				Name: "test_tool",
				Parameters: map[string]string{
					"param0": "input",
				},
				Function: func(input string) string {
					return "tool result: " + input
				},
			},
		})
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Set up memory to track the conversation
		mem := shorttermmemory.New()

		// Setup initial user message
		userMsg := messages.Message[messages.UserMessage]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.UserMessage{
				Content: messages.ContentOrParts{
					Content: "Test parallel tool calls",
				},
			},
			Sender:    "user",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddUserPrompt(userMsg)

		// Setup tool call message
		toolCallMsg := messages.Message[messages.ToolCallMessage]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "tool1",
						Name:      "test_tool",
						Arguments: `{"input":"test input"}`,
					},
				},
			},
			Sender:    "test_agent",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddToolCall(toolCallMsg)

		// Setup tool response message
		toolResponseMsg := messages.Message[messages.ToolResponse]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.ToolResponse{
				ToolCallID: "tool1",
				ToolName:   "test_tool",
				Content:    "tool result: test input",
			},
			Sender:    "test_tool",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddToolResponse(toolResponseMsg)

		// Setup final assistant message
		assistantMsg := messages.Message[messages.AssistantMessage]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "final result with tool: tool result: test input",
				},
			},
			Sender:    "test_agent",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddAssistantMessage(assistantMsg)

		// Setup provider expectations for tool call
		toolCallEvents := make(chan provider.StreamEvent, 1)
		toolCallEvents <- provider.Response[messages.ToolCallMessage]{
			RunID:      runID,
			TurnID:     turnID,
			Checkpoint: mem.Checkpoint(),
			Response:   toolCallMsg.Payload,
		}
		close(toolCallEvents)

		// Setup provider expectations for final response
		finalEvents := make(chan provider.StreamEvent, 1)
		finalEvents <- provider.Response[messages.AssistantMessage]{
			RunID:      runID,
			TurnID:     turnID,
			Checkpoint: mem.Checkpoint(),
			Response:   assistantMsg.Payload,
		}
		close(finalEvents)

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(toolCallEvents, nil).Once()

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(finalEvents, nil).Once()

		// Setup topic expectations for events
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(3)

		// Expect tool call response
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Response[messages.ToolCallMessage])
			return ok && resp.TurnID == turnID
		})).Return(nil).Once()

		// Expect tool response
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Request[messages.ToolResponse])
			return ok && resp.Message.ToolCallID == "tool1"
		})).Return(nil).Once()

		// Expect final assistant message
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Response[messages.AssistantMessage])
			return ok && resp.Response.Content.Content == "final result with tool: tool result: test input"
		})).Return(nil).Once()

		var result string
		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:              "test_agent",
				Model:             "test_model",
				ParallelToolCalls: false,
			},
			MaxTurns:   10,
			Checkpoint: mem.Checkpoint(),
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		require.NoError(t, env.env.GetWorkflowResult(&result))
		assert.Equal(t, "final result with tool: tool result: test input", result)
	})

	t.Run("parallel tool calls", func(t *testing.T) {
		env := setupTestEnvironment(t)

		// Register workflow and activities
		env.env.RegisterWorkflow(env.temporal.Run)
		env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
		env.env.RegisterActivity(env.temporal.RunCompletion)
		env.env.RegisterActivity(env.temporal.CallTool)

		agent := mocks.NewAgent(t)
		prov := mocks.NewProvider(t)
		model := mocks.NewModel(t)

		runID := uuidx.New()
		turnID := uuidx.New()

		// Register mock agent in the global registry
		agent.EXPECT().Name().Return("test_agent")
		agent.EXPECT().Tools().Return([]tool.Definition{
			{
				Name: "test_tool1",
				Parameters: map[string]string{
					"param0": "input", // Add parameter mapping
				},
				Function: func(input string) string {
					return "tool1 result: " + input
				},
			},
			{
				Name: "test_tool2",
				Parameters: map[string]string{
					"param0": "input", // Add parameter mapping
				},
				Function: func(input string) string {
					return "tool2 result: " + input
				},
			},
		})
		buboagent.Add(agent)

		// Set up and register mock model
		model.EXPECT().Name().Return("test_model")
		model.EXPECT().Provider().Return(prov)
		models.Add(model)

		// Clean up registries after test
		t.Cleanup(func() {
			buboagent.Del("test_agent")
			models.Del("test_model")
		})

		// Set up memory to track the conversation
		mem := shorttermmemory.New()

		// Setup initial user message
		userMsg := messages.Message[messages.UserMessage]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.UserMessage{
				Content: messages.ContentOrParts{
					Content: "Test parallel tool calls",
				},
			},
			Sender:    "user",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddUserPrompt(userMsg)

		// Setup tool call message
		toolCallMsg := messages.Message[messages.ToolCallMessage]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.ToolCallMessage{
				ToolCalls: []messages.ToolCallData{
					{
						ID:        "tool1",
						Name:      "test_tool1",
						Arguments: `{"input":"input1"}`,
					},
					{
						ID:        "tool2",
						Name:      "test_tool2",
						Arguments: `{"input":"input2"}`,
					},
				},
			},
			Sender:    "test_agent",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddToolCall(toolCallMsg)

		// Setup tool response messages
		toolResponse1Msg := messages.Message[messages.ToolResponse]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.ToolResponse{
				ToolCallID: "tool1",
				ToolName:   "test_tool1",
				Content:    "tool1 result: input1",
			},
			Sender:    "test_tool1",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddToolResponse(toolResponse1Msg)

		toolResponse2Msg := messages.Message[messages.ToolResponse]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.ToolResponse{
				ToolCallID: "tool2",
				ToolName:   "test_tool2",
				Content:    "tool2 result: input2",
			},
			Sender:    "test_tool2",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddToolResponse(toolResponse2Msg)

		// Setup final assistant message
		assistantMsg := messages.Message[messages.AssistantMessage]{
			RunID:  runID,
			TurnID: turnID,
			Payload: messages.AssistantMessage{
				Content: messages.AssistantContentOrParts{
					Content: "final result with tools: tool1 result: input1, tool2 result: input2",
				},
			},
			Sender:    "test_agent",
			Timestamp: strfmt.DateTime(time.Now()),
		}
		mem.AddAssistantMessage(assistantMsg)

		// Setup provider expectations for parallel tool calls
		toolCallEvents := make(chan provider.StreamEvent, 1)
		toolCallEvents <- provider.Response[messages.ToolCallMessage]{
			RunID:      runID,
			TurnID:     turnID,
			Checkpoint: mem.Checkpoint(),
			Response:   toolCallMsg.Payload,
		}
		close(toolCallEvents)

		// Setup provider expectations for final response
		finalEvents := make(chan provider.StreamEvent, 1)
		finalEvents <- provider.Response[messages.AssistantMessage]{
			RunID:      runID,
			TurnID:     turnID,
			Checkpoint: mem.Checkpoint(),
			Response:   assistantMsg.Payload,
		}
		close(finalEvents)

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(toolCallEvents, nil).Once()

		prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
			return p.RunID == runID
		})).Return(finalEvents, nil).Once()

		// Setup topic expectations
		mockTopic := mocks.NewTopic(t)
		env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(4)

		// Expect tool calls response
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Response[messages.ToolCallMessage])
			return ok && resp.TurnID == turnID
		})).Return(nil).Once()

		// Expect tool1 response
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Request[messages.ToolResponse])
			return ok && resp.Message.ToolCallID == "tool1"
		})).Return(nil).Once()

		// Expect tool2 response
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Request[messages.ToolResponse])
			return ok && resp.Message.ToolCallID == "tool2"
		})).Return(nil).Once()

		// Expect final assistant message
		mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
			evt, ok := msg.(events.Event)
			if !ok {
				return false
			}
			resp, ok := evt.(events.Response[messages.AssistantMessage])
			return ok && resp.Response.Content.Content == "final result with tools: tool1 result: input1, tool2 result: input2"
		})).Return(nil).Once()

		var result string
		env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
			ID: runID,
			Agent: RemoteAgent{
				Name:              "test_agent",
				Model:             "test_model",
				ParallelToolCalls: true,
			},
			MaxTurns:   10,
			Checkpoint: mem.Checkpoint(),
		})

		require.True(t, env.env.IsWorkflowCompleted())
		require.NoError(t, env.env.GetWorkflowError())
		require.NoError(t, env.env.GetWorkflowResult(&result))
		assert.Equal(t, "final result with tools: tool1 result: input1, tool2 result: input2", result)
	})
}

func TestTemporalMaxTurns(t *testing.T) {
	env := setupTestEnvironment(t)

	// Register workflow and activities
	env.env.RegisterWorkflow(env.temporal.Run)
	env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
	env.env.RegisterActivity(env.temporal.RunCompletion)
	env.env.RegisterActivity(env.temporal.CallTool)
	env.env.RegisterActivity(env.temporal.PublishError)

	agent := mocks.NewAgent(t)
	prov := mocks.NewProvider(t)
	model := mocks.NewModel(t)

	runID := uuidx.New()

	// Register mock agent in the global registry
	agent.EXPECT().Name().Return("test_agent")
	agent.EXPECT().Tools().Return([]tool.Definition{
		{
			Description: "test tool",
			Name:        "test_tool",
			Parameters: map[string]string{
				"p0": "arg",
			},
			Function: func(arg string) string {
				return "tool result: " + arg
			},
		},
	}) // No tools available
	buboagent.Add(agent)

	// Set up and register mock model
	model.EXPECT().Name().Return("test_model")
	model.EXPECT().Provider().Return(prov)
	models.Add(model)

	// Clean up registries after test
	t.Cleanup(func() {
		buboagent.Del("test_agent")
		models.Del("test_model")
	})

	// Setup provider expectations for both turns
	turnID1 := uuidx.New()
	turnID2 := uuidx.New()

	// First turn: return tool call
	firstTurnEvents := make(chan provider.StreamEvent, 1)
	firstTurnEvents <- provider.Response[messages.ToolCallMessage]{
		RunID:  runID,
		TurnID: turnID1,
		Response: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					ID:        "tool1",
					Name:      "test_tool",
					Arguments: `{"arg":"test input"}`,
				},
			},
		},
	}
	close(firstTurnEvents)

	// Second turn: return tool call again
	secondTurnEvents := make(chan provider.StreamEvent, 1)
	secondTurnEvents <- provider.Response[messages.ToolCallMessage]{
		RunID:  runID,
		TurnID: turnID2,
		Response: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					ID:        "tool2",
					Name:      "test_tool",
					Arguments: `{"arg":"test input 2"}`,
				},
			},
		},
	}
	close(secondTurnEvents)

	// Setup provider expectations
	prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
		return p.RunID == runID
	})).Return(firstTurnEvents, nil).Once()

	prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
		return p.RunID == runID
	})).Return(secondTurnEvents, nil).Once()

	// Setup topic expectations for events
	mockTopic := mocks.NewTopic(t)
	env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(5) // 2 tool calls + 2 tool responses + 1 error

	// Expect first tool call response
	mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
		evt, ok := msg.(events.Event)
		if !ok {
			return false
		}
		resp, ok := evt.(events.Response[messages.ToolCallMessage])
		return ok && resp.TurnID == turnID1
	})).Return(nil).Once()

	// Expect first tool response
	mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
		evt, ok := msg.(events.Event)
		if !ok {
			return false
		}
		resp, ok := evt.(events.Request[messages.ToolResponse])
		return ok && resp.Message.ToolCallID == "tool1"
	})).Return(nil).Once()

	// Expect second tool call response
	mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
		evt, ok := msg.(events.Event)
		if !ok {
			return false
		}
		resp, ok := evt.(events.Response[messages.ToolCallMessage])
		return ok && resp.TurnID == turnID2
	})).Return(nil).Once()

	// Expect second tool response
	mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
		evt, ok := msg.(events.Event)
		if !ok {
			return false
		}
		resp, ok := evt.(events.Request[messages.ToolResponse])
		return ok && resp.Message.ToolCallID == "tool2"
	})).Return(nil).Once()

	// Expect max turns error
	mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
		evt, ok := msg.(events.Event)
		if !ok {
			return false
		}
		errEvt, ok := evt.(events.Error)
		return ok && strings.Contains(errEvt.Err.Error(), "max turns reached")
	})).Return(nil).Once()

	env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
		ID: runID,
		Agent: RemoteAgent{
			Name:  "test_agent",
			Model: "test_model",
		},
		MaxTurns: 2,
	})

	require.True(t, env.env.IsWorkflowCompleted())
	require.Error(t, env.env.GetWorkflowError())
	assert.Contains(t, env.env.GetWorkflowError().Error(), "max turns reached")
}

func TestTemporalRunWithAgentChain(t *testing.T) {
	env := setupTestEnvironment(t)

	// Register workflows and activities
	env.env.RegisterWorkflow(env.temporal.Run)
	env.env.RegisterWorkflow(env.temporal.RunChildWorkflow)
	env.env.RegisterActivity(env.temporal.RunCompletion)
	env.env.RegisterActivity(env.temporal.CallTool)

	// Create and register agents
	agent1 := mocks.NewAgent(t)
	agent2 := mocks.NewAgent(t)
	prov := mocks.NewProvider(t)
	model := mocks.NewModel(t)

	// Set up agent1 expectations
	agent1.EXPECT().Name().Return("agent1").Times(1) // Called for registration, initial completion, tool call, and final completion
	agent1.EXPECT().Tools().Return([]tool.Definition{
		{
			Name: "transfer_to_agent2",
			Function: func() api.Agent {
				return agent2
			},
		},
	}).Once() // Called during tool call
	buboagent.Add(agent1)

	// Set up agent2 expectations
	agent2.EXPECT().Name().Return("agent2").Times(3)                    // Called for registration, completion, and child workflow
	agent2.EXPECT().Model().Return(model).Times(1)                      // Called for registration and completion
	agent2.EXPECT().Instructions().Return("agent2 instructions").Once() // Called during completion
	agent2.EXPECT().ParallelToolCalls().Return(false).Once()            // Called during completion
	buboagent.Add(agent2)

	// Set up model expectations
	model.EXPECT().Name().Return("test_model").Times(2) // Called for registration (2) and completion (3)
	model.EXPECT().Provider().Return(prov).Times(3)     // Called for registration (2) and completion (3)
	models.Add(model)

	// Clean up registries after test
	t.Cleanup(func() {
		buboagent.Del("agent1")
		buboagent.Del("agent2")
		models.Del("test_model")
	})

	runID := uuidx.New()

	// Set up initial memory with user message
	mem := shorttermmemory.New()
	turnID := uuidx.New()
	userMsg := messages.Message[messages.UserMessage]{
		RunID:  runID,
		TurnID: turnID,
		Payload: messages.UserMessage{
			Content: messages.ContentOrParts{
				Content: "Start the agent chain",
			},
		},
		Sender:    "user",
		Timestamp: strfmt.DateTime(time.Now()),
	}
	mem.AddUserPrompt(userMsg)

	// Set up tool call message for agent1
	toolCallMsg := messages.Message[messages.ToolCallMessage]{
		RunID:  runID,
		TurnID: turnID,
		Payload: messages.ToolCallMessage{
			ToolCalls: []messages.ToolCallData{
				{
					ID:        "transfer1",
					Name:      "transfer_to_agent2",
					Arguments: "{}",
				},
			},
		},
		Sender:    "agent1",
		Timestamp: strfmt.DateTime(time.Now()),
	}
	mem.AddToolCall(toolCallMsg)

	agent1Events := make(chan provider.StreamEvent, 1)
	agent1Events <- provider.Response[messages.ToolCallMessage]{
		RunID:      runID,
		TurnID:     turnID,
		Checkpoint: mem.Checkpoint(),
		Response:   toolCallMsg.Payload,
	}
	close(agent1Events)

	// Set up provider responses for agent2
	assistantMsg := messages.Message[messages.AssistantMessage]{
		RunID:  runID,
		TurnID: turnID,
		Payload: messages.AssistantMessage{
			Content: messages.AssistantContentOrParts{
				Content: "final result",
			},
		},
		Sender:    "agent2",
		Timestamp: strfmt.DateTime(time.Now()),
	}
	mem.AddAssistantMessage(assistantMsg)

	agent2Events := make(chan provider.StreamEvent, 1)
	agent2Events <- provider.Response[messages.AssistantMessage]{
		RunID:      runID,
		TurnID:     turnID,
		Checkpoint: mem.Checkpoint(),
		Response:   assistantMsg.Payload,
	}
	close(agent2Events)

	// Set up provider expectations for all completions
	prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
		return p.RunID == runID && p.Instructions == "agent1 instructions"
	})).Return(agent1Events, nil).Times(2) // Called for initial and final completion

	prov.EXPECT().ChatCompletion(mock.Anything, mock.MatchedBy(func(p provider.CompletionParams) bool {
		return p.RunID == runID && p.Instructions == "agent2 instructions"
	})).Return(agent2Events, nil).Once() // Called for agent2 completion

	// Set up topic for event publishing
	mockTopic := mocks.NewTopic(t)
	env.broker.EXPECT().Topic(mock.Anything, runID.String()).Return(mockTopic).Times(2) // Called for initial completion, tool call, child workflow completion, and final completion
	mockTopic.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(msg interface{}) bool {
		_, ok := msg.(events.Event)
		return ok
	})).Return(nil).Times(2) // Called for initial completion, tool call, child workflow completion, and final completion

	var result string
	env.env.ExecuteWorkflow(env.temporal.Run, RemoteRunCommand{
		ID: runID,
		Agent: RemoteAgent{
			Name:         "agent1",
			Model:        "test_model",
			Instructions: "agent1 instructions",
		},
		MaxTurns:   10,
		Checkpoint: mem.Checkpoint(),
	})

	require.True(t, env.env.IsWorkflowCompleted())
	require.NoError(t, env.env.GetWorkflowError())
	require.NoError(t, env.env.GetWorkflowResult(&result))
	assert.Equal(t, "final result", result)
}
