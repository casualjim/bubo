package stdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZero(t *testing.T) {
	t.Run("numeric types", func(t *testing.T) {
		assert.Equal(t, 0, Zero[int]())
		assert.Equal(t, int8(0), Zero[int8]())
		assert.Equal(t, int16(0), Zero[int16]())
		assert.Equal(t, int32(0), Zero[int32]())
		assert.Equal(t, int64(0), Zero[int64]())
		assert.Equal(t, uint(0), Zero[uint]())
		assert.Equal(t, uint8(0), Zero[uint8]())
		assert.Equal(t, uint16(0), Zero[uint16]())
		assert.Equal(t, uint32(0), Zero[uint32]())
		assert.Equal(t, uint64(0), Zero[uint64]())
		assert.Equal(t, float32(0), Zero[float32]())
		assert.Equal(t, float64(0), Zero[float64]())
	})

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, "", Zero[string]())
	})

	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, false, Zero[bool]())
	})

	t.Run("slice", func(t *testing.T) {
		var expected []int
		assert.Equal(t, expected, Zero[[]int]())
	})

	t.Run("map", func(t *testing.T) {
		var expected map[string]int
		assert.Equal(t, expected, Zero[map[string]int]())
	})

	t.Run("channel", func(t *testing.T) {
		var expected chan int
		assert.Equal(t, expected, Zero[chan int]())
	})

	t.Run("pointer", func(t *testing.T) {
		var expected *int
		assert.Equal(t, expected, Zero[*int]())
	})

	t.Run("interface", func(t *testing.T) {
		var expected interface{}
		assert.Equal(t, expected, Zero[interface{}]())
	})

	t.Run("struct", func(t *testing.T) {
		type testStruct struct {
			A int
			B string
			C bool
			D []int
			E map[string]int
		}
		expected := testStruct{}
		assert.Equal(t, expected, Zero[testStruct]())
	})
}
