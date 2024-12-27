package reflectx

import (
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

func IsFunction(fn any) bool {
	if fn == nil {
		return false
	}

	ftpe := reflect.TypeOf(fn)
	isFunc := ftpe.Kind() == reflect.Func

	return isFunc
}

func FunctionName(fn any) string {
	if !IsFunction(fn) {
		return ""
	}

	val := reflect.ValueOf(fn)
	typ := val.Type()

	var name string
	// For named types (like AgentFunction), use the type name
	if typ.Name() != "" {
		name = typ.String()
	} else {
		// For methods, use the method name
		if typ.NumIn() > 0 && typ.In(0).Kind() == reflect.Struct {
			if fn := runtime.FuncForPC(val.Pointer()); fn != nil {
				name = fn.Name()
				if lastDot := strings.LastIndex(name, "."); lastDot >= 0 {
					name = strings.TrimSuffix(name[lastDot+1:], "-fm")
				}
			}
		} else {
			// For anonymous functions, use the full signature
			if fn := runtime.FuncForPC(val.Pointer()); fn != nil {
				name = fn.Name()
				if lastDot := strings.LastIndex(name, "."); lastDot >= 0 {
					name = strings.TrimSuffix(name[lastDot+1:], "-fm")
				}
			} else {
				name = typ.String()
			}
		}
	}
	return name
}

// methodPattern matches method expressions in the following formats:
// 1. Optional package path: github.com/example/pkg.
// 2. Type name with optional pointer and parentheses:
//   - (*Type).Method - pointer receiver with parentheses
//   - (Type).Method - value receiver with parentheses
//   - Type.Method - value receiver without parentheses
//
// 3. Method name
//
// The pattern captures:
// - Group 1: The type name without decorators
// - Group 2: The method name
var (
	methodPattern = `^(?:[^(]*?\.)?(?:\(\*([^)]+)\)|\(([^)]+)\)|([^.(]+))\.(\w+)$`
	methodRegex   = regexp.MustCompile(methodPattern)
)

// IsStructMethod checks if the provided value is a method expression (e.g., (*Type).Method or Type.Method).
// It distinguishes between method expressions and regular functions that take struct parameters.
//
// A method expression is created by qualifying a method with a type name, like (*T).Method or T.Method.
// This is different from:
// - Regular functions that take struct parameters
// - Method values (like t.Method where t is an instance)
// - Anonymous functions
//
// The function works by:
// 1. Verifying the input is a function with at least one parameter
// 2. Checking that the first parameter is a struct type (or pointer to struct)
// 3. Using regex to identify method expressions and extract the type name
// 4. Comparing the extracted type name with the actual struct type
func IsStructMethod(f any) bool {
	if f == nil {
		return false
	}

	t := reflect.TypeOf(f)

	// Must be a function
	if t.Kind() != reflect.Func {
		return false
	}

	// Must have at least one parameter (the receiver)
	if t.NumIn() == 0 {
		return false
	}

	// Get the first parameter type (potential receiver)
	firstParam := t.In(0)

	// For pointer receiver methods
	for firstParam.Kind() == reflect.Ptr {
		firstParam = firstParam.Elem()
	}

	// Must be a struct type
	if firstParam.Kind() != reflect.Struct {
		return false
	}

	// Get the function name from runtime
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()

	// Match against the method pattern
	matches := methodRegex.FindStringSubmatch(funcName)
	if matches == nil {
		return false
	}

	// Get the struct type name
	structName := firstParam.Name()

	// Check if any of the type name groups match the struct name
	// matches[1]: pointer receiver type name
	// matches[2]: value receiver type name with parentheses
	// matches[3]: value receiver type name without parentheses
	for i := 1; i <= 3; i++ {
		if matches[i] == structName {
			return true
		}
	}

	return false
}
