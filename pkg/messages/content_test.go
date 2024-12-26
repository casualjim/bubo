package messages

import (
	"strings"
	"sync"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentOrParts_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		content ContentOrParts
		want    string
		wantErr bool
	}{
		{
			name:    "empty content and parts",
			content: ContentOrParts{},
			want:    "null",
		},
		{
			name: "simple string content",
			content: ContentOrParts{
				Content: "hello world",
			},
			want: `"hello world"`,
		},
		{
			name: "whitespace only content should marshal to parts",
			content: ContentOrParts{
				Content: "   ",
			},
			want: "null",
		},
		{
			name: "single text part",
			content: ContentOrParts{
				Parts: []ContentPart{
					TextContentPart{Text: "hello world"},
				},
			},
			want: `[{"type":"text","text":"hello world"}]`,
		},
		{
			name: "single image part",
			content: ContentOrParts{
				Parts: []ContentPart{
					ImageContentPart{URL: "http://example.com/image.jpg"},
				},
			},
			want: `[{"type":"image","image_url":"http://example.com/image.jpg"}]`,
		},
		{
			name: "single audio part",
			content: ContentOrParts{
				Parts: []ContentPart{
					AudioContentPart{InputAudio: InputAudio{Data: []byte("test audio data"), Format: "mp3"}},
				},
			},
			want: `[{"type":"audio","input_audio":{"data":"dGVzdCBhdWRpbyBkYXRh","format":"mp3"}}]`,
		},
		{
			name: "multiple mixed parts",
			content: ContentOrParts{
				Parts: []ContentPart{
					TextContentPart{Text: "hello"},
					ImageContentPart{URL: "http://example.com/image.jpg"},
					AudioContentPart{InputAudio: InputAudio{Data: []byte("test audio data"), Format: "mp3"}},
				},
			},
			want: `[{"type":"text","text":"hello"},{"type":"image","image_url":"http://example.com/image.jpg"},{"type":"audio","input_audio":{"data":"dGVzdCBhdWRpbyBkYXRh","format":"mp3"}}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))

			// Test round-trip
			var decoded ContentOrParts
			err = json.Unmarshal(got, &decoded)
			require.NoError(t, err)
			reencoded, err := json.Marshal(decoded)
			require.NoError(t, err)
			require.JSONEq(t, string(got), string(reencoded))
		})
	}
}

func TestContentOrParts_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ContentOrParts
		wantErr bool
	}{
		{
			name:    "invalid json",
			input:   `{invalid json`,
			wantErr: true,
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  ContentOrParts{Parts: []ContentPart{}},
		},
		{
			name:  "simple string content",
			input: `"hello world"`,
			want: ContentOrParts{
				Content: "hello world",
			},
		},
		{
			name:  "empty string content",
			input: `""`,
			want:  ContentOrParts{Content: ""},
		},
		{
			name:  "single text part",
			input: `[{"type":"text","text":"hello world"}]`,
			want: ContentOrParts{
				Parts: []ContentPart{
					TextContentPart{Text: "hello world"},
				},
			},
		},
		{
			name:  "single image part",
			input: `[{"type":"image","image_url":"http://example.com/image.jpg"}]`,
			want: ContentOrParts{
				Parts: []ContentPart{
					ImageContentPart{URL: "http://example.com/image.jpg"},
				},
			},
		},
		{
			name:  "single audio part",
			input: `[{"type":"audio","input_audio":{"data":"dGVzdCBhdWRpbyBkYXRh","format":"mp3"}}]`,
			want: ContentOrParts{
				Parts: []ContentPart{
					AudioContentPart{InputAudio: InputAudio{Data: []byte("test audio data"), Format: "mp3"}},
				},
			},
		},
		{
			name:    "unknown part type",
			input:   `[{"type":"unknown","data":"something"}]`,
			wantErr: true,
		},
		{
			name:    "invalid text part",
			input:   `[{"type":"text"}]`,
			wantErr: true,
		},
		{
			name:    "invalid image part",
			input:   `[{"type":"image"}]`,
			wantErr: true,
		},
		{
			name:    "invalid audio part",
			input:   `[{"type":"audio"}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ContentOrParts
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// For array cases, compare the marshaled results
			if len(tt.want.Parts) > 0 {
				gotJSON, err := json.Marshal(got)
				require.NoError(t, err)
				wantJSON, err := json.Marshal(tt.want)
				require.NoError(t, err)
				assert.JSONEq(t, string(wantJSON), string(gotJSON))
			} else {
				assert.Equal(t, tt.want.Content, got.Content)
			}
		})
	}
}

func TestAssistantContentOrParts_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		content AssistantContentOrParts
		want    string
		wantErr bool
	}{
		{
			name:    "empty content and parts",
			content: AssistantContentOrParts{},
			want:    "null",
		},
		{
			name: "simple string content",
			content: AssistantContentOrParts{
				Content: "hello world",
			},
			want: `"hello world"`,
		},
		{
			name: "single text part",
			content: AssistantContentOrParts{
				Parts: []AssistantContentPart{
					TextContentPart{Text: "hello world"},
				},
			},
			want: `[{"type":"text","text":"hello world"}]`,
		},
		{
			name: "single refusal part",
			content: AssistantContentOrParts{
				Parts: []AssistantContentPart{
					RefusalContentPart{Refusal: "I cannot help with that"},
				},
			},
			want: `[{"type":"refusal","refusal":"I cannot help with that"}]`,
		},
		{
			name: "multiple mixed parts",
			content: AssistantContentOrParts{
				Parts: []AssistantContentPart{
					TextContentPart{Text: "hello"},
					RefusalContentPart{Refusal: "I cannot help with that"},
				},
			},
			want: `[{"type":"text","text":"hello"},{"type":"refusal","refusal":"I cannot help with that"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))

			// Test round-trip
			var decoded AssistantContentOrParts
			err = json.Unmarshal(got, &decoded)
			require.NoError(t, err)
			reencoded, err := json.Marshal(decoded)
			require.NoError(t, err)
			assert.JSONEq(t, string(got), string(reencoded))
		})
	}
}

func TestAssistantContentOrParts_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AssistantContentOrParts
		wantErr bool
	}{
		{
			name:    "invalid json",
			input:   `{invalid json`,
			wantErr: true,
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  AssistantContentOrParts{Parts: []AssistantContentPart{}},
		},
		{
			name:  "simple string content",
			input: `"hello world"`,
			want: AssistantContentOrParts{
				Content: "hello world",
			},
		},
		{
			name:  "single text part",
			input: `[{"type":"text","text":"hello world"}]`,
			want: AssistantContentOrParts{
				Parts: []AssistantContentPart{
					TextContentPart{Text: "hello world"},
				},
			},
		},
		{
			name:  "single refusal part",
			input: `[{"type":"refusal","refusal":"I cannot help with that"}]`,
			want: AssistantContentOrParts{
				Parts: []AssistantContentPart{
					RefusalContentPart{Refusal: "I cannot help with that"},
				},
			},
		},
		{
			name:    "unknown part type",
			input:   `[{"type":"unknown","data":"something"}]`,
			wantErr: true,
		},
		{
			name:    "invalid text part",
			input:   `[{"type":"text"}]`,
			wantErr: true,
		},
		{
			name:    "invalid refusal part",
			input:   `[{"type":"refusal"}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AssistantContentOrParts
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// For array cases, compare the marshaled results
			if len(tt.want.Parts) > 0 {
				gotJSON, err := json.Marshal(got)
				require.NoError(t, err)
				wantJSON, err := json.Marshal(tt.want)
				require.NoError(t, err)
				assert.JSONEq(t, string(wantJSON), string(gotJSON))
			} else {
				assert.Equal(t, tt.want.Content, got.Content)
			}
		})
	}
}

// Individual content part tests
func TestTextContentPart(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TextContentPart
		wantErr bool
	}{
		{
			name:  "valid text part",
			input: `{"type":"text","text":"hello world"}`,
			want:  TextContentPart{Text: "hello world"},
		},
		{
			name:  "empty text",
			input: `{"type":"text","text":""}`,
			want:  TextContentPart{Text: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got TextContentPart
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			// Test round-trip
			marshaled, err := got.MarshalJSON()
			require.NoError(t, err)
			var unmarshaled TextContentPart
			err = json.Unmarshal(marshaled, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, got, unmarshaled)
		})
	}
}

func TestImageContentPart(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ImageContentPart
		wantErr bool
	}{
		{
			name:  "valid image part",
			input: `{"type":"image","image_url":"http://example.com/image.jpg"}`,
			want:  ImageContentPart{URL: "http://example.com/image.jpg"},
		},
		{
			name:  "empty URL",
			input: `{"type":"image","image_url":""}`,
			want:  ImageContentPart{URL: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ImageContentPart
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			// Test round-trip
			marshaled, err := got.MarshalJSON()
			require.NoError(t, err)
			var unmarshaled ImageContentPart
			err = json.Unmarshal(marshaled, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, got, unmarshaled)
		})
	}
}

func TestAudioContentPart(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AudioContentPart
		wantErr bool
	}{
		{
			name:  "valid audio part",
			input: `{"type":"audio","input_audio":{"data":"dGVzdCBhdWRpbyBkYXRh","format":"mp3"}}`,
			want: AudioContentPart{
				InputAudio: InputAudio{
					Data:   []byte("test audio data"),
					Format: "mp3",
				},
			},
		},
		{
			name:  "empty audio data",
			input: `{"type":"audio","input_audio":{"data":"","format":""}}`,
			want: AudioContentPart{
				InputAudio: InputAudio{
					Data:   []byte{},
					Format: "",
				},
			},
		},
		{
			name:    "invalid input_audio structure",
			input:   `{"type":"audio","input_audio":"invalid"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AudioContentPart
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			// Test round-trip
			marshaled, err := got.MarshalJSON()
			require.NoError(t, err)
			var unmarshaled AudioContentPart
			err = json.Unmarshal(marshaled, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, got, unmarshaled)
		})
	}
}

func TestRefusalContentPart(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    RefusalContentPart
		wantErr bool
	}{
		{
			name:  "valid refusal part",
			input: `{"type":"refusal","refusal":"I cannot help with that"}`,
			want:  RefusalContentPart{Refusal: "I cannot help with that"},
		},
		{
			name:  "empty refusal",
			input: `{"type":"refusal","refusal":""}`,
			want:  RefusalContentPart{Refusal: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got RefusalContentPart
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			// Test round-trip
			marshaled, err := got.MarshalJSON()
			require.NoError(t, err)
			var unmarshaled RefusalContentPart
			err = json.Unmarshal(marshaled, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, got, unmarshaled)
		})
	}
}

// Interface implementation tests
func TestInterfaceImplementations(t *testing.T) {
	t.Run("ContentPart interface", func(t *testing.T) {
		// Verify that types implement ContentPart interface
		var _ ContentPart = TextContentPart{}
		var _ ContentPart = ImageContentPart{}
		var _ ContentPart = AudioContentPart{}
	})

	t.Run("AssistantContentPart interface", func(t *testing.T) {
		// Verify that types implement AssistantContentPart interface
		var _ AssistantContentPart = TextContentPart{}
		var _ AssistantContentPart = RefusalContentPart{}
	})
}

// Edge cases and special scenarios
func TestEdgeCases(t *testing.T) {
	t.Run("very large content", func(t *testing.T) {
		// Create a large string
		largeString := strings.Repeat("a", 1<<20) // 1MB string
		content := ContentOrParts{Content: largeString}

		data, err := json.Marshal(content)
		require.NoError(t, err)

		var decoded ContentOrParts
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, content.Content, decoded.Content)
	})

	t.Run("special characters in content", func(t *testing.T) {
		specialChars := "Hello\n\t\"'\\世界"
		content := ContentOrParts{Content: specialChars}

		data, err := json.Marshal(content)
		require.NoError(t, err)

		var decoded ContentOrParts
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, content.Content, decoded.Content)
	})

	t.Run("empty parts slice", func(t *testing.T) {
		content := ContentOrParts{Parts: make([]ContentPart, 0)}
		data, err := json.Marshal(content)
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("nil parts slice", func(t *testing.T) {
		content := ContentOrParts{}
		data, err := json.Marshal(content)
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})

	t.Run("concurrent access", func(t *testing.T) {
		content := ContentOrParts{
			Parts: []ContentPart{
				TextContentPart{Text: "hello"},
				ImageContentPart{URL: "http://example.com/image.jpg"},
			},
		}

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := json.Marshal(content)
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
	})
}

// Benchmark tests
func BenchmarkContentOrParts(b *testing.B) {
	content := ContentOrParts{
		Parts: []ContentPart{
			TextContentPart{Text: "hello"},
			ImageContentPart{URL: "http://example.com/image.jpg"},
			AudioContentPart{InputAudio: InputAudio{Data: []byte("test audio data"), Format: "mp3"}},
		},
	}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(content)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	data, _ := json.Marshal(content)
	b.Run("Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var decoded ContentOrParts
			err := json.Unmarshal(data, &decoded)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
