package reflectx

import (
	"io"
	"reflect"
	"testing"
)

type resultTestInterface interface {
	ResultTestMethod() string
}

type resultImplementer struct{}

func (i resultImplementer) ResultTestMethod() string { return "test" }

type resultNonImplementer struct{}

func TestResultImplements(t *testing.T) {
	t.Run("function returning interface implementer", func(t *testing.T) {
		fn := func() resultImplementer { return resultImplementer{} }
		if !ResultImplements[resultTestInterface](fn) {
			t.Error("expected function to implement resultTestInterface")
		}
	})

	t.Run("function returning non-implementer", func(t *testing.T) {
		fn := func() resultNonImplementer { return resultNonImplementer{} }
		if ResultImplements[resultTestInterface](fn) {
			t.Error("expected function to not implement resultTestInterface")
		}
	})

	t.Run("function with multiple returns, one implements", func(t *testing.T) {
		fn := func() (int, resultImplementer, error) { return 0, resultImplementer{}, nil }
		if !ResultImplements[resultTestInterface](fn) {
			t.Error("expected function to implement resultTestInterface")
		}
	})

	t.Run("function with multiple returns, none implements", func(t *testing.T) {
		fn := func() (int, string, error) { return 0, "", nil }
		if ResultImplements[resultTestInterface](fn) {
			t.Error("expected function to not implement resultTestInterface")
		}
	})

	t.Run("nil function", func(t *testing.T) {
		if ResultImplements[resultTestInterface](nil) {
			t.Error("expected nil function to return false")
		}
	})

	t.Run("non-function input", func(t *testing.T) {
		if ResultImplements[resultTestInterface]("not a function") {
			t.Error("expected non-function to return false")
		}
	})

	t.Run("real world example with io.Reader", func(t *testing.T) {
		fn := func() (*testing.T, io.Reader, error) { return nil, nil, nil }
		if !ResultImplements[io.Reader](fn) {
			t.Error("expected function to implement io.Reader")
		}
	})

	t.Run("reflect.Type input", func(t *testing.T) {
		fn := func() resultImplementer { return resultImplementer{} }
		fnType := reflect.TypeOf(fn)
		if !ResultImplements[resultTestInterface](fnType) {
			t.Error("expected reflect.Type to work with ResultImplements")
		}
	})
}
