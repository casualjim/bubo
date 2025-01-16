package jsonx

import "encoding/json"

// ToDynamicJSON converts any Go value to a dynamic JSON object represented as a map[string]any.
// It first marshals the input value to JSON bytes and then unmarshals those bytes into a map.
// If either the marshaling or unmarshaling process fails, an error is returned.
//
// Parameters:
//   - val: The input value of any type to be converted to a dynamic JSON object.
//
// Returns:
//   - map[string]any: A map representing the dynamic JSON object.
//   - error: An error if the conversion fails, otherwise nil.
func ToDynamicJSON(val any) (map[string]any, error) {
	result := make(map[string]any)
	b, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}
