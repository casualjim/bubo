package reflectx

import (
	"reflect"
)

// ResultImplements checks if any of the result arguments of a function implements the given interface type T.
// It returns true if at least one result argument implements the interface, false otherwise.
// The function parameter can be either a function value or a reflect.Type of a function.
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
