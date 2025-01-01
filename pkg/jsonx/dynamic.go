package jsonx

import "encoding/json"

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
