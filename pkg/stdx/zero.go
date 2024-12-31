package stdx

// Zero returns the zero value for a given type T.
//
// The zero value is the default value that variables of type T
// are initialized to when declared without an explicit initializer.
// For example:
//   - For numeric types (int, float, etc.), the zero value is 0
//   - For strings, the zero value is ""
//   - For pointers, interfaces, channels, maps, and slices, the zero value is nil
//   - For structs, the zero value has all fields set to their respective zero values
func Zero[T any]() T {
	var zero T
	return zero
}
