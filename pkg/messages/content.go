// Package messages provides types and functionality for handling multi-part message content
// in different formats including text, images, and audio.
package messages

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	json "github.com/goccy/go-json"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var jsonNull = []byte(`null`)

// ContentOrParts represents either a simple string content or a collection of content parts.
// It provides flexible serialization to handle both single-string messages and multi-part content.
type ContentOrParts struct {
	Content string        // Raw string content, used when the message is just text
	Parts   []ContentPart // Slice of different content parts (text, image, audio)
	_       struct{}      // require keyed usage
}

// MarshalJSON implements json.Marshaler interface for ContentOrParts.
// Returns the Content as a JSON string if it's non-empty,
// otherwise returns the Parts as a JSON array.
// Returns null if both Content and Parts are empty.
func (c ContentOrParts) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(c.Content) != "" {
		return json.Marshal(c.Content)
	}
	if c.Parts == nil {
		return jsonNull, nil
	}
	return json.Marshal(c.Parts)
}

// UnmarshalJSON implements json.Unmarshaler interface for ContentOrParts.
// Handles both string content and arrays of different content part types (text, image, audio).
// Returns an error if the JSON is invalid or contains unknown content part types.
func (c *ContentOrParts) UnmarshalJSON(input []byte) error {
	if !gjson.ValidBytes(input) {
		return fmt.Errorf("invalid json: %s", input)
	}
	jv := gjson.ParseBytes(input)
	if jv.IsArray() {
		aj := jv.Array()
		parts := make([]ContentPart, len(aj))
		for idx, ajv := range aj {
			tpe := ajv.Get("type").String()
			switch tpe {
			case "text":
				var part TextContentPart
				if err := part.UnmarshalJSON([]byte(ajv.Raw)); err != nil {
					return fmt.Errorf("invalid text part at %d: %w", idx, err)
				}
				parts[idx] = part
			case "image":
				var part ImageContentPart
				if err := part.UnmarshalJSON([]byte(ajv.Raw)); err != nil {
					return fmt.Errorf("invalid image part at %d: %w", idx, err)
				}
				parts[idx] = part
			case "audio":
				var part AudioContentPart
				if err := part.UnmarshalJSON([]byte(ajv.Raw)); err != nil {
					return fmt.Errorf("invalid audio part at %d: %w", idx, err)
				}
				parts[idx] = part
			default:
				return fmt.Errorf("content part at %d has an unknown type %q", idx, tpe)
			}
		}
		c.Parts = parts
		return nil
	}
	c.Content = jv.String()
	return nil
}

// AssistantContentOrParts represents content that can be either a simple string
// or a collection of assistant-specific content parts (text or refusal).
type AssistantContentOrParts struct {
	Content string                 // Raw string content for simple text responses
	Parts   []AssistantContentPart // Slice of assistant-specific content parts
	Refusal string
	_       struct{} // require keyed usage
}

// MarshalJSON implements json.Marshaler interface for AssistantContentOrParts.
// Returns the Content as a JSON string if it's non-empty,
// otherwise returns the Parts as a JSON array.
// Returns null if both Content and Parts are empty.
func (c AssistantContentOrParts) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(c.Content) != "" && strings.TrimSpace(c.Refusal) != "" {
		return nil, fmt.Errorf("both Content and Refusal are set")
	}
	if strings.TrimSpace(c.Content) != "" {
		return json.Marshal(c.Content)
	}
	if strings.TrimSpace(c.Refusal) != "" {
		return json.Marshal(c.Refusal)
	}
	if c.Parts == nil {
		return jsonNull, nil
	}
	return json.Marshal(c.Parts)
}

// UnmarshalJSON implements json.Unmarshaler interface for AssistantContentOrParts.
// Handles both string content and arrays of assistant-specific content parts (text, refusal).
// Returns an error if the JSON is invalid or contains unknown content part types.
func (c *AssistantContentOrParts) UnmarshalJSON(input []byte) error {
	if !gjson.ValidBytes(input) {
		return fmt.Errorf("invalid json: %s", input)
	}
	jv := gjson.ParseBytes(input)
	if jv.IsArray() {
		aj := jv.Array()
		parts := make([]AssistantContentPart, len(aj))
		for idx, ajv := range aj {
			tpe := ajv.Get("type").String()
			switch tpe {
			case "text":
				var part TextContentPart
				if err := part.UnmarshalJSON([]byte(ajv.Raw)); err != nil {
					return fmt.Errorf("invalid assistant text part at %d: %w", idx, err)
				}
				parts[idx] = part
			case "refusal":
				var part RefusalContentPart
				if err := part.UnmarshalJSON([]byte(ajv.Raw)); err != nil {
					return fmt.Errorf("invalid assistant refusal part at %d: %w", idx, err)
				}
				parts[idx] = part
			default:
				return fmt.Errorf("content part at %d has an unknown type %q", idx, tpe)
			}
		}
		c.Parts = parts
		return nil
	}
	c.Content = jv.String()
	return nil
}

// ContentPart is an interface that marks structs as valid content parts.
// Implementations include TextContentPart, ImageContentPart, and AudioContentPart.
type ContentPart interface {
	contentPart()
}

// AssistantContentPart is an interface that marks structs as valid assistant content parts.
// Implementations include TextContentPart and RefusalContentPart.
type AssistantContentPart interface {
	assistantContentPart()
}

// Text creates a new TextContentPart with the given text.
// This is a convenience function for creating text content parts.
func Text(text string) TextContentPart {
	return TextContentPart{Text: text}
}

// TextContentPart represents a text-only content part.
// It implements both ContentPart and AssistantContentPart interfaces.
type TextContentPart struct {
	Text string   `json:"text"` // The actual text content
	_    struct{} // require keyed usage
}

func (TextContentPart) contentPart()          {}
func (TextContentPart) assistantContentPart() {}

var tcpJSON = []byte(`{"type":"text"}`)

// MarshalJSON implements json.Marshaler interface for TextContentPart.
// Serializes the text content with a "type":"text" field.
func (t TextContentPart) MarshalJSON() ([]byte, error) {
	return sjson.SetBytes(tcpJSON, "text", t.Text)
}

// UnmarshalJSON implements json.Unmarshaler interface for TextContentPart.
// Validates and extracts the required 'text' field from the JSON input.
func (t *TextContentPart) UnmarshalJSON(input []byte) error {
	text := gjson.GetBytes(input, "text")
	if !text.Exists() {
		return errors.New("missing required field 'text'")
	}
	t.Text = text.String()
	return nil
}

// Refusal creates a new RefusalContentPart with the given reason.
// This is a convenience function for creating refusal messages.
func Refusal(reason string) RefusalContentPart {
	return RefusalContentPart{Refusal: reason}
}

// RefusalContentPart represents a content part indicating a refusal message.
// It implements the AssistantContentPart interface.
type RefusalContentPart struct {
	Refusal string   `json:"refusal"` // The refusal message text
	_       struct{} // require keyed usage
}

func (RefusalContentPart) assistantContentPart() {}

var rcpJSON = []byte(`{"type":"refusal"}`)

// MarshalJSON implements json.Marshaler interface for RefusalContentPart.
// Serializes the refusal content with a "type":"refusal" field.
func (t RefusalContentPart) MarshalJSON() ([]byte, error) {
	return sjson.SetBytes(rcpJSON, "refusal", t.Refusal)
}

// UnmarshalJSON implements json.Unmarshaler interface for RefusalContentPart.
// Validates and extracts the required 'refusal' field from the JSON input.
func (t *RefusalContentPart) UnmarshalJSON(input []byte) error {
	refusal := gjson.GetBytes(input, "refusal")
	if !refusal.Exists() {
		return errors.New("missing required field 'refusal'")
	}
	t.Refusal = refusal.String()
	return nil
}

// Image creates a new ImageContentPart with the given URL.
// This is a convenience function for creating image content parts.
func Image(url string) ImageContentPart {
	return ImageContentPart{URL: url}
}

// ImageContentPart represents an image content part with a URL.
// It implements the ContentPart interface.
type ImageContentPart struct {
	URL string   `json:"image_url"` // URL pointing to the image
	_   struct{} // require keyed usage
}

func (ImageContentPart) contentPart() {}

var icpJSON = []byte(`{"type":"image"}`)

// MarshalJSON implements json.Marshaler interface for ImageContentPart.
// Serializes the image URL with a "type":"image" field.
func (i ImageContentPart) MarshalJSON() ([]byte, error) {
	return sjson.SetBytes(icpJSON, "image_url", i.URL)
}

// UnmarshalJSON implements json.Unmarshaler interface for ImageContentPart.
// Validates and extracts the required 'image_url' field from the JSON input.
func (i *ImageContentPart) UnmarshalJSON(input []byte) error {
	uri := gjson.GetBytes(input, "image_url")
	if !uri.Exists() {
		return errors.New("missing required field 'image_url")
	}
	i.URL = uri.String()
	return nil
}

// InputAudio contains the data and format information for audio content.
type InputAudio struct {
	Data   []byte   `json:"-"`      // Raw audio data
	Format string   `json:"format"` // Audio format specification
	_      struct{} // require keyed usage
}

// MarshalJSON implements json.Marshaler interface for InputAudio.
// Encodes the Data field as base64 in the JSON output.
func (i InputAudio) MarshalJSON() ([]byte, error) {
	type Alias InputAudio
	return json.Marshal(&struct {
		Data string `json:"data"`
		*Alias
	}{
		Data:  base64.StdEncoding.EncodeToString(i.Data),
		Alias: (*Alias)(&i),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface for InputAudio.
// Decodes the base64 encoded data field from JSON into the Data byte slice.
func (i *InputAudio) UnmarshalJSON(data []byte) error {
	type Alias InputAudio
	aux := &struct {
		Data string `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(i),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	i.Data, err = base64.StdEncoding.DecodeString(aux.Data)
	return err
}

// Audio creates a new AudioContentPart with the provided raw audio data and format.
// The format parameter should specify the audio encoding format (e.g., "wav", "mp3").
// This is a convenience function for creating audio content parts.
func Audio(data []byte, format string) ContentPart {
	return AudioContentPart{
		InputAudio: InputAudio{
			Data:   data,
			Format: format,
		},
	}
}

// AudioContentPart represents an audio content part.
// It implements the ContentPart interface.
type AudioContentPart struct {
	InputAudio InputAudio `json:"input_audio"` // Audio data and format information
	_          struct{}   // require keyed usage
}

func (AudioContentPart) contentPart() {}

var acpJSON = []byte(`{"type":"audio"}`)

// MarshalJSON implements json.Marshaler interface for AudioContentPart.
// Serializes the audio input data and format with a "type":"audio" field.
func (i AudioContentPart) MarshalJSON() ([]byte, error) {
	jj, err := json.Marshal(i.InputAudio)
	if err != nil {
		return nil, err
	}
	return sjson.SetRawBytes(acpJSON, "input_audio", jj)
}

// UnmarshalJSON implements json.Unmarshaler interface for AudioContentPart.
// Validates and extracts the required 'input_audio' object containing 'data' and 'format' fields.
func (i *AudioContentPart) UnmarshalJSON(input []byte) error {
	if !gjson.ValidBytes(input) {
		return fmt.Errorf("invalid json for audio part")
	}

	audioJson := gjson.GetBytes(input, "input_audio")
	if !audioJson.Exists() {
		return fmt.Errorf("missing required field 'input_audio'")
	}

	if !audioJson.IsObject() {
		return fmt.Errorf("'input_audio' must be an object")
	}

	data := audioJson.Get("data")
	format := audioJson.Get("format")

	if !data.Exists() || !format.Exists() {
		return fmt.Errorf("input_audio requires both 'data' and 'format' fields")
	}

	decodedData, err := base64.StdEncoding.DecodeString(data.String())
	if err != nil {
		return fmt.Errorf("invalid base64 data: %w", err)
	}

	i.InputAudio = InputAudio{
		Data:   decodedData,
		Format: format.String(),
	}

	return nil
}
