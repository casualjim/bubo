package reflectx

import (
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

type functionTestStruct struct{}

func (t *functionTestStruct) method() {}
func (t functionTestStruct) method2() {}

func regularFunction()   {}
func withParams(x int)   {}
func withReturn() error  { return nil }
func variadic(...string) {}

func TestFunctionValidation(t *testing.T) {
	tests := []struct {
		name string
		fn   interface{}
		want bool
	}{
		{"nil", nil, false},
		{"int", 42, false},
		{"string", "not a func", false},
		{"struct", functionTestStruct{}, false},
		{"regular function", regularFunction, true},
		{"anonymous function", func() {}, true},
		{"function with params", withParams, true},
		{"function with return", withReturn, true},
		{"variadic function", variadic, true},
		{"pointer method", (*functionTestStruct).method, true},
		{"value method", (functionTestStruct).method2, true},
		{"function with multiple params", func(a, b string) {}, true},
		{"function with multiple returns", func() (int, error) { return 0, nil }, true},
		{"complex function", func(x int) (string, error) { return "", nil }, true},
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsFunction(tt.fn))
		})
	}
}

type testInterface interface {
	Do()
}

type methodTestStruct struct{}

func (m *methodTestStruct) pointerMethod()              {}
func (m methodTestStruct) valueMethod()                 {}
func (m *methodTestStruct) pointerMethodWithArgs(x int) {}
func (m methodTestStruct) valueMethodWithReturn() error { return nil }

func takesPointer(x *string)                 {}
func takesStruct(x methodTestStruct)         {}
func takesInterface(x testInterface)         {}
func takesStructPointer(x *methodTestStruct) {}

func TestFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		fn       interface{}
		expected string
	}{
		{"nil", nil, ""},
		{"int", 42, ""},
		{"string", "not a func", ""},
		{"regular function", regularFunction, "regularFunction"},
		{"function with params", withParams, "withParams"},
		{"function with return", withReturn, "withReturn"},
		{"variadic function", variadic, "variadic"},
		{"pointer method", (*methodTestStruct).pointerMethod, "pointerMethod"},
		{"value method", (methodTestStruct).valueMethod, "valueMethod"},
		{"method with args", (*methodTestStruct).pointerMethodWithArgs, "pointerMethodWithArgs"},
		{"method with return", (methodTestStruct).valueMethodWithReturn, "valueMethodWithReturn"},
		{"anonymous function", func() {}, ""}, // Empty string comparison skipped in test
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			got := FunctionName(tt.fn)
			if tt.name == "anonymous function" {
				// For anonymous functions, just verify we get a non-empty name
				require.NotEmpty(t, got, "expected non-empty name for anonymous function")
				return
			}
			require.Equal(t, tt.expected, got)
		})
	}
}

type (
	M  map[string]any
	MM map[string]any
)

func TestIsRefinedType(t *testing.T) {
	tests := []struct {
		name     string
		typeFunc func() reflect.Type
		want     bool
	}{
		{
			name: "same named type M",
			typeFunc: func() reflect.Type {
				var m M
				return reflect.TypeOf(m)
			},
			want: true,
		},
		{
			name: "different named type MM",
			typeFunc: func() reflect.Type {
				var m MM
				return reflect.TypeOf(m)
			},
			want: false,
		},
		{
			name: "plain map type",
			typeFunc: func() reflect.Type {
				var m map[string]any
				return reflect.TypeOf(m)
			},
			want: false,
		},
		{
			name: "non-map type string",
			typeFunc: func() reflect.Type {
				var s string
				return reflect.TypeOf(s)
			},
			want: false,
		},
		{
			name: "non-map type struct",
			typeFunc: func() reflect.Type {
				var s struct{}
				return reflect.TypeOf(s)
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRefinedType[M](tt.typeFunc())
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMethodValidation(t *testing.T) {
	tests := []struct {
		name string
		fn   interface{}
		want bool
	}{
		{"nil", nil, false},
		{"regular function", regularFunction, false},
		{"pointer method", (*methodTestStruct).pointerMethod, true},
		{"triple pointer method", (*methodTestStruct).pointerMethod, true},
		{"value method", (methodTestStruct).valueMethod, true},
		{"pointer method with args", (*methodTestStruct).pointerMethodWithArgs, true},
		{"value method with return", (methodTestStruct).valueMethodWithReturn, true},
		{"function taking pointer", takesPointer, false},
		{"function taking struct", takesStruct, false},
		{"function taking interface", takesInterface, false},
		{"function taking struct pointer", takesStructPointer, false},
		{"anonymous function with struct pointer", func(s *methodTestStruct) {}, false},
		{"method expression pointer", (&methodTestStruct{}).pointerMethod, false},
		{"method expression value", (methodTestStruct{}).valueMethod, false},
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsStructMethod(tt.fn))
		})
	}
}
