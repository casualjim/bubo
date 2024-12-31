package reflectx

import (
	"testing"
)

type (
	customString string
	customInt    int
)

type testStruct struct {
	Name   string
	Age    int
	Active bool
}

// Generic struct with single type parameter
type Generic[T any] struct {
	Value T
	Extra string
}

// Generic struct with multiple type parameters
type MultiGeneric[K comparable, V any] struct {
	Key   K
	Value V
}

// Generic struct with nested generic type
type NestedGeneric[T any] struct {
	Inner Generic[T]
	Data  []T
}

// Deeply nested generic struct
type DeepNested[T comparable, U any] struct {
	First  NestedGeneric[T]
	Second Generic[U]
	Pair   MultiGeneric[T, U]
}

// Generic interface
type GenericValuer[T any] interface {
	Value() T
}

// Implementation of generic interface
type genericImpl[T any] struct {
	val T
}

func (g genericImpl[T]) Value() T {
	return g.val
}

type nestedStruct struct {
	Basic     testStruct
	Pointer   *testStruct
	Slice     []string
	Map       map[string]int
	Interface interface{}
}

type implementer struct {
	value string
}

func (i implementer) String() string {
	return i.value
}

func TestIsZero(t *testing.T) {
	nonNilPtr := &testStruct{}
	var nilPtr *testStruct
	var nilIface interface{}
	var nilImpl implementer
	nonNilImpl := implementer{value: "test"}

	// Generic type instances
	var zeroGenericInt Generic[int]
	var zeroGenericString Generic[string]
	nonZeroGenericInt := Generic[int]{Value: 42, Extra: "test"}

	var zeroMultiGeneric MultiGeneric[string, int]
	nonZeroMultiGeneric := MultiGeneric[string, int]{Key: "key", Value: 42}

	var zeroNestedGeneric NestedGeneric[int]
	nonZeroNestedGeneric := NestedGeneric[int]{
		Inner: Generic[int]{Value: 42},
		Data:  []int{1, 2, 3},
	}

	var zeroDeepNested DeepNested[int, string]
	nonZeroDeepNested := DeepNested[int, string]{
		First:  NestedGeneric[int]{Inner: Generic[int]{Value: 42}},
		Second: Generic[string]{Value: "test"},
		Pair:   MultiGeneric[int, string]{Key: 42, Value: "test"},
	}

	// Generic interface instances
	var nilGenericValuer GenericValuer[int]
	var zeroGenericImpl genericImpl[int]
	nonZeroGenericImpl := genericImpl[int]{val: 42}

	tests := []struct {
		name string
		v    interface{}
		want bool
	}{
		// Nil values
		{"nil interface", nil, true},
		{"nil pointer", nilPtr, true},
		{"nil interface value", nilIface, true},

		// Basic types - zero values
		{"zero int", 0, true},
		{"zero string", "", true},
		{"zero bool", false, true},
		{"zero float", 0.0, true},
		{"zero complex", complex(0, 0), true},

		// Basic types - non-zero values
		{"non-zero int", 42, false},
		{"non-zero string", "hello", false},
		{"non-zero bool", true, false},
		{"non-zero float", 3.14, false},
		{"non-zero complex", complex(1, 2), false},

		// Custom basic types
		{"zero custom string", customString(""), true},
		{"non-zero custom string", customString("hello"), false},
		{"zero custom int", customInt(0), true},
		{"non-zero custom int", customInt(42), false},

		// Generic structs - single type parameter
		{"zero Generic[int]", zeroGenericInt, true},
		{"zero Generic[string]", zeroGenericString, true},
		{"non-zero Generic[int]", nonZeroGenericInt, false},
		{"partially filled Generic[int]", Generic[int]{Value: 42}, false},

		// Generic structs - multiple type parameters
		{"zero MultiGeneric", zeroMultiGeneric, true},
		{"non-zero MultiGeneric", nonZeroMultiGeneric, false},

		// Nested generic structs
		{"zero NestedGeneric", zeroNestedGeneric, true},
		{"non-zero NestedGeneric", nonZeroNestedGeneric, false},
		{"partially filled NestedGeneric", NestedGeneric[int]{Inner: Generic[int]{Value: 42}}, false},

		// Deeply nested generic structs
		{"zero DeepNested", zeroDeepNested, true},
		{"non-zero DeepNested", nonZeroDeepNested, false},
		{"partially filled DeepNested", DeepNested[int, string]{
			First: NestedGeneric[int]{Inner: Generic[int]{Value: 42}},
		}, false},

		// Generic interfaces
		{"nil GenericValuer", nilGenericValuer, true},
		{"zero genericImpl", zeroGenericImpl, true},
		{"non-zero genericImpl", nonZeroGenericImpl, false},

		// Pointers to generic types
		{"nil Generic pointer", (*Generic[int])(nil), true},
		{"non-nil zero Generic pointer", &Generic[int]{}, true},
		{"non-nil non-zero Generic pointer", &Generic[int]{Value: 42}, false},

		// Pointers
		{"nil struct pointer", nilPtr, true},
		{"non-nil zero struct pointer", nonNilPtr, true},
		{"non-nil non-zero struct pointer", &testStruct{Name: "test"}, false},

		// Slices
		{"nil slice", []int(nil), true},
		{"empty slice", []int{}, false},
		{"non-empty slice", []int{1, 2, 3}, false},

		// Maps
		{"nil map", map[string]int(nil), true},
		{"empty map", map[string]int{}, false},
		{"non-empty map", map[string]int{"one": 1}, false},

		// Structs
		{"zero struct", testStruct{}, true},
		{"partially filled struct", testStruct{Name: "test"}, false},
		{"fully filled struct", testStruct{Name: "test", Age: 30, Active: true}, false},

		// Nested structs
		{"zero nested struct", nestedStruct{}, true},
		// A struct containing a field, even if that field is zero, makes the parent struct non-zero
		{"nested struct with zero basic", nestedStruct{Basic: testStruct{}}, true},
		// A struct containing an explicitly set nil pointer is still non-zero
		{"nested struct with nil pointer", nestedStruct{Pointer: nil}, true},
		{"nested struct with non-nil pointer", nestedStruct{Pointer: &testStruct{}}, false},

		// Interfaces
		{"nil interface implementer", nilImpl, true},
		{"non-nil interface implementer", nonNilImpl, false},
		{"interface with nil value", interface{}(nil), true},
		{"interface with non-nil value", interface{}("test"), false},

		// Arrays
		{"zero array", [3]int{}, true},
		{"non-zero array", [3]int{1, 2, 3}, false},

		// Channels
		{"nil channel", (chan int)(nil), true},
		{"non-nil channel", make(chan int), false},

		// Functions
		{"nil func", (func())(nil), true},
		{"non-nil func", func() {}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsZero(tt.v); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsZeroEdgeCases tests some special edge cases
func TestIsZeroEdgeCases(t *testing.T) {
	// Test deeply nested pointers
	var deepPtr ***string
	if !IsZero(deepPtr) {
		t.Error("IsZero() failed for deeply nested nil pointer")
	}

	str := "test"
	ptr1 := &str
	ptr2 := &ptr1
	ptr3 := &ptr2
	if IsZero(ptr3) {
		t.Error("IsZero() failed for deeply nested non-nil pointer")
	}

	// Test circular references
	type circular struct {
		self *circular
	}
	c := &circular{}
	c.self = c
	if IsZero(c) {
		t.Error("IsZero() failed for circular reference")
	}

	// Test interface containing nil
	var nilPtr *int
	iface := interface{}(nilPtr)
	if !IsZero(iface) {
		t.Error("IsZero() failed for interface containing nil pointer")
	}

	// Test generic circular references
	type genericCircular[T any] struct {
		value T
		self  *genericCircular[T]
	}
	gc := &genericCircular[int]{value: 42}
	gc.self = gc
	if IsZero(gc) {
		t.Error("IsZero() failed for generic circular reference")
	}

	// Test nested generic interfaces
	type nestedInterface[T any] interface {
		Get() Generic[T]
	}
	var nilNestedIface nestedInterface[int]
	if !IsZero(nilNestedIface) {
		t.Error("IsZero() failed for nil nested generic interface")
	}
}
