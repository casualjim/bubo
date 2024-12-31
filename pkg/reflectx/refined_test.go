package reflectx

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	CustomString string
	CustomInt    int
	CustomStruct struct {
		Field string
	}
)

func TestIsRefinedTypeWithPrimitives(t *testing.T) {
	tests := []struct {
		name     string
		value    reflect.Type
		typeFunc func() bool
		expected bool
	}{
		{
			name:  "matching string type",
			value: reflect.TypeOf(""),
			typeFunc: func() bool {
				return IsRefinedType[string](reflect.TypeOf(""))
			},
			expected: true,
		},
		{
			name:  "non-matching string type",
			value: reflect.TypeOf(CustomString("")),
			typeFunc: func() bool {
				return IsRefinedType[string](reflect.TypeOf(CustomString("")))
			},
			expected: false,
		},
		{
			name:  "matching custom string type",
			value: reflect.TypeOf(CustomString("")),
			typeFunc: func() bool {
				return IsRefinedType[CustomString](reflect.TypeOf(CustomString("")))
			},
			expected: true,
		},
		{
			name:  "matching int type",
			value: reflect.TypeOf(0),
			typeFunc: func() bool {
				return IsRefinedType[int](reflect.TypeOf(0))
			},
			expected: true,
		},
		{
			name:  "non-matching int type",
			value: reflect.TypeOf(CustomInt(0)),
			typeFunc: func() bool {
				return IsRefinedType[int](reflect.TypeOf(CustomInt(0)))
			},
			expected: false,
		},
		{
			name:  "matching custom int type",
			value: reflect.TypeOf(CustomInt(0)),
			typeFunc: func() bool {
				return IsRefinedType[CustomInt](reflect.TypeOf(CustomInt(0)))
			},
			expected: true,
		},
		{
			name:  "matching struct type",
			value: reflect.TypeOf(CustomStruct{}),
			typeFunc: func() bool {
				return IsRefinedType[CustomStruct](reflect.TypeOf(CustomStruct{}))
			},
			expected: true,
		},
		{
			name:  "matching pointer type",
			value: reflect.TypeOf(&CustomStruct{}),
			typeFunc: func() bool {
				return IsRefinedType[*CustomStruct](reflect.TypeOf(&CustomStruct{}))
			},
			expected: true,
		},
		{
			name:  "non-matching pointer vs value type",
			value: reflect.TypeOf(&CustomStruct{}),
			typeFunc: func() bool {
				return IsRefinedType[CustomStruct](reflect.TypeOf(&CustomStruct{}))
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typeFunc()
			assert.Equal(t, tt.expected, result)
		})
	}
}
