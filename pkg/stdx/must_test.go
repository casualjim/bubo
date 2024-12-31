package stdx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errTest = errors.New("test error")

func TestMust0(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Must0(nil)
		})
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must0(errTest)
		})
	})
}

func TestMust1(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		result := Must1("test", nil)
		assert.Equal(t, "test", result)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must1("test", errTest)
		})
	})
}

func TestMust2(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		v1, v2 := Must2("test", 42, nil)
		assert.Equal(t, "test", v1)
		assert.Equal(t, 42, v2)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must2("test", 42, errTest)
		})
	})
}

func TestMust3(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		v1, v2, v3 := Must3("test", 42, true, nil)
		assert.Equal(t, "test", v1)
		assert.Equal(t, 42, v2)
		assert.Equal(t, true, v3)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must3("test", 42, true, errTest)
		})
	})
}

func TestMust4(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		v1, v2, v3, v4 := Must4("test", 42, true, 3.14, nil)
		assert.Equal(t, "test", v1)
		assert.Equal(t, 42, v2)
		assert.Equal(t, true, v3)
		assert.Equal(t, 3.14, v4)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must4("test", 42, true, 3.14, errTest)
		})
	})
}

func TestMust5(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		v1, v2, v3, v4, v5 := Must5("test", 42, true, 3.14, []int{1}, nil)
		assert.Equal(t, "test", v1)
		assert.Equal(t, 42, v2)
		assert.Equal(t, true, v3)
		assert.Equal(t, 3.14, v4)
		assert.Equal(t, []int{1}, v5)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must5("test", 42, true, 3.14, []int{1}, errTest)
		})
	})
}

func TestMust6(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		v1, v2, v3, v4, v5, v6 := Must6("test", 42, true, 3.14, []int{1}, "six", nil)
		assert.Equal(t, "test", v1)
		assert.Equal(t, 42, v2)
		assert.Equal(t, true, v3)
		assert.Equal(t, 3.14, v4)
		assert.Equal(t, []int{1}, v5)
		assert.Equal(t, "six", v6)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must6("test", 42, true, 3.14, []int{1}, "six", errTest)
		})
	})
}

func TestMust7(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		v1, v2, v3, v4, v5, v6, v7 := Must7("test", 42, true, 3.14, []int{1}, "six", 7.0, nil)
		assert.Equal(t, "test", v1)
		assert.Equal(t, 42, v2)
		assert.Equal(t, true, v3)
		assert.Equal(t, 3.14, v4)
		assert.Equal(t, []int{1}, v5)
		assert.Equal(t, "six", v6)
		assert.Equal(t, 7.0, v7)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must7("test", 42, true, 3.14, []int{1}, "six", 7.0, errTest)
		})
	})
}

func TestMust8(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		v1, v2, v3, v4, v5, v6, v7, v8 := Must8("test", 42, true, 3.14, []int{1}, "six", 7.0, uint(8), nil)
		assert.Equal(t, "test", v1)
		assert.Equal(t, 42, v2)
		assert.Equal(t, true, v3)
		assert.Equal(t, 3.14, v4)
		assert.Equal(t, []int{1}, v5)
		assert.Equal(t, "six", v6)
		assert.Equal(t, 7.0, v7)
		assert.Equal(t, uint(8), v8)
	})

	t.Run("with error", func(t *testing.T) {
		assert.PanicsWithError(t, errTest.Error(), func() {
			Must8("test", 42, true, 3.14, []int{1}, "six", 7.0, uint(8), errTest)
		})
	})
}
