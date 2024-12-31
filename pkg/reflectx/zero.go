package reflectx

import "reflect"

// IsZero checks if a value is the zero value for its type.
// It handles nil values, pointers, interfaces, and all other types.
// Returns true if the value is zero or nil, false otherwise.
func IsZero(v any) bool {
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)

	// Handle pointers and interfaces
	switch val.Kind() {
	case reflect.Ptr, reflect.Interface:
		if val.IsNil() {
			return true
		}
		return IsZero(val.Elem().Interface())
	}

	return val.IsZero()
}
