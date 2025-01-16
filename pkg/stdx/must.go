package stdx

// Must0 is a helper function that panics if the provided error is not nil.
// It is intended to be used for error handling in situations where an error
// is not expected and should cause the program to terminate if it occurs.
//
// Parameters:
//   - err: The error to check. If it is not nil, the function will panic.
func Must0(err error) {
	if err != nil {
		panic(err)
	}
}

// Must1 is a generic function that takes a value of any type and an error.
// If the error is not nil, it panics with the error. Otherwise, it returns the value.
//
// This function is useful for simplifying error handling in cases where you
// are confident that an error will not occur, or where you want to handle
// errors by panicking.
//
// T: The type of the value to be returned.
// v: The value to be returned if err is nil.
// err: The error to check.
func Must1[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Must2 is a helper function that takes two values of any type and an error.
// If the error is not nil, it panics with the error. Otherwise, it returns the two values.
//
// This function is useful for handling functions that return multiple values and an error,
// allowing you to avoid explicit error handling in cases where you are confident that the error
// will not occur.
//
// Example usage:
//
//	result1, result2 := Must2(someFunctionThatReturnsTwoValuesAndError())
//
// T: The type of the first value.
// V: The type of the second value.
func Must2[T any, V any](t T, v V, err error) (T, V) {
	if err != nil {
		panic(err)
	}
	return t, v
}

// Must3 is a helper function that takes three values of any type and an error.
// If the error is not nil, it panics with the error. Otherwise, it returns the three values.
//
// This function is useful for handling multiple return values from functions that
// return an error as one of the values.
//
// Parameters:
//   - t: The first value of any type.
//   - v: The second value of any type.
//   - u: The third value of any type.
//   - err: The error to check.
//
// Returns:
//   - The first value (t) if err is nil.
//   - The second value (v) if err is nil.
//   - The third value (u) if err is nil.
//
// Panics:
//   - If err is not nil, it panics with the provided error.
func Must3[T any, V any, U any](t T, v V, u U, err error) (T, V, U) {
	if err != nil {
		panic(err)
	}
	return t, v, u
}

// Must4 is a helper function that takes four values of any type and an error.
// If the error is not nil, it panics with the error. Otherwise, it returns the four values.
//
// This function is useful for handling multiple return values from functions
// that return an error as one of the values.
//
// Parameters:
//   - t: The first value of any type.
//   - v: The second value of any type.
//   - u: The third value of any type.
//   - w: The fourth value of any type.
//   - err: The error to check.
//
// Returns:
//   - The four input values (t, v, u, w) if err is nil.
//
// Panics:
//   - If err is not nil.
func Must4[T any, V any, U any, W any](t T, v V, u U, w W, err error) (T, V, U, W) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w
}

// Must5 is a helper function that takes five values of any type and an error.
// If the error is not nil, it panics with the error.
// Otherwise, it returns the five values.
//
// This function is useful for handling multiple return values from functions
// that return an error as one of their return values.
//
// Parameters:
//   - t: The first value of any type.
//   - v: The second value of any type.
//   - u: The third value of any type.
//   - w: The fourth value of any type.
//   - x: The fifth value of any type.
//   - err: The error to check.
//
// Returns:
//   - The five input values (t, v, u, w, x) if err is nil.
//
// Panics:
//   - If err is not nil, it panics with the provided error.
func Must5[T any, V any, U any, W any, X any](t T, v V, u U, w W, x X, err error) (T, V, U, W, X) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x
}

// Must6 takes six input values of any type along with an error.
// If the error is not nil, it panics with the provided error.
// Otherwise, it returns the six input values.
//
// Parameters:
//   - t: The first input value of any type.
//   - v: The second input value of any type.
//   - u: The third input value of any type.
//   - w: The fourth input value of any type.
//   - x: The fifth input value of any type.
//   - y: The sixth input value of any type.
//   - err: An error that is checked to determine if a panic should occur.
//
// Returns:
//   - The six input values (t, v, u, w, x, y) if err is nil.
func Must6[T, V, U, W, X, Y any](t T, v V, u U, w W, x X, y Y, err error) (T, V, U, W, X, Y) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x, y
}

// Must7 is a generic function that takes seven values of any type and an error.
// If the error is not nil, it panics with the error. Otherwise, it returns the seven values.
//
// Parameters:
//   - t: The first value of any type.
//   - v: The second value of any type.
//   - u: The third value of any type.
//   - w: The fourth value of any type.
//   - x: The fifth value of any type.
//   - y: The sixth value of any type.
//   - z: The seventh value of any type.
//   - err: The error to check.
//
// Returns:
//   - The seven input values (t, v, u, w, x, y, z) if err is nil.
//
// Panics:
//   - If err is not nil, it panics with the provided error.
func Must7[T, V, U, W, X, Y, Z any](t T, v V, u U, w W, x X, y Y, z Z, err error) (T, V, U, W, X, Y, Z) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x, y, z
}

// Must8 is a utility function that takes eight values of any type and an error.
// If the error is not nil, it panics with the error. Otherwise, it returns the
// eight values.
//
// This function is useful for handling multiple return values from functions
// that return an error as one of their return values.
//
// Parameters:
//   - t: The first value of any type.
//   - v: The second value of any type.
//   - u: The third value of any type.
//   - w: The fourth value of any type.
//   - x: The fifth value of any type.
//   - y: The sixth value of any type.
//   - z: The seventh value of any type.
//   - a: The eighth value of any type.
//   - err: The error to check.
//
// Returns:
//   - The eight values (t, v, u, w, x, y, z, a) if err is nil.
//
// Panics:
//   - If err is not nil, it panics with the provided error.
func Must8[T, V, U, W, X, Y, Z, A any](t T, v V, u U, w W, x X, y Y, z Z, a A, err error) (T, V, U, W, X, Y, Z, A) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x, y, z, a
}
