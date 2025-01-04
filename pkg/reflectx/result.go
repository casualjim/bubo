package reflectx

import (
	"reflect"
)

// ResultImplements checks if any of the result arguments of a function implements the given interface type T.
// This is particularly useful for runtime type checking and reflection-based functionality where you need
// to verify function return types against interfaces.
//
// Design choices:
//   - Generic implementation allows compile-time type safety for the interface type
//   - Accepts both function values and reflect.Type for flexibility
//   - Returns false for nil or non-function inputs rather than panicking
//   - Uses reflect.TypeOf(&zero).Elem() pattern to get interface type to handle both concrete and interface types
//
// Example usage:
//
//	type Stringer interface { String() string }
//
//	func returnsStringer() fmt.Stringer { return nil }
//	func returnsMultiple() (int, fmt.Stringer, error) { return 0, nil, nil }
//
//	ResultImplements[fmt.Stringer](returnsStringer)    // returns true
//	ResultImplements[fmt.Stringer](returnsMultiple)    // returns true
//	ResultImplements[error](returnsMultiple)           // returns true
//	ResultImplements[fmt.Stringer](func() int { })     // returns false
//
// Parameters:
//   - function: Either a function value or reflect.Type of a function. If nil or not a function, returns false.
//
// Returns:
//   - bool: true if any result argument implements interface T, false otherwise
func ResultImplements[T any](function interface{}) bool {
	if function == nil {
		return false
	}

	// Get the type of the function
	var fnType reflect.Type
	switch v := function.(type) {
	case reflect.Type:
		if v.Kind() != reflect.Func {
			return false
		}
		fnType = v
	default:
		fnType = reflect.TypeOf(function)
		if fnType.Kind() != reflect.Func {
			return false
		}
	}

	// Get the interface type using the generic parameter
	var zero T
	ifaceType := reflect.TypeOf(&zero).Elem()

	// Check each result argument
	for i := 0; i < fnType.NumOut(); i++ {
		resultType := fnType.Out(i)
		if resultType.Implements(ifaceType) {
			return true
		}
	}

	return false
}
