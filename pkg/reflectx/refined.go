package reflectx

import (
	"reflect"
)

// IsRefinedType checks if the provided reflect.Type matches the type of the generic parameter R.
//
// Parameters:
// - value: The reflect.Type to be checked.
//
// Returns:
// - bool: True if the type of the generic parameter R matches the provided reflect.Type, otherwise false.
func IsRefinedType[R any](value reflect.Type) bool {
	var toMatch R
	mt := reflect.TypeOf(toMatch)

	isRefinedType := mt == value
	return isRefinedType
}
