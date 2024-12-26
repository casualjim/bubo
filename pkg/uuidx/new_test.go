package uuidx

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Test that New() returns a valid UUID
	id := New()
	assert.Equal(t, uuid.Version(7), id.Version(), "UUID should be version 7")
	assert.Equal(t, uuid.RFC4122, id.Variant(), "UUID should have RFC4122 variant")

	// Test uniqueness
	id2 := New()
	assert.NotEqual(t, id, id2, "Generated UUIDs should be unique")
}

func TestNewString(t *testing.T) {
	// Test that NewString() returns a valid UUID string
	idStr := NewString()
	id, err := uuid.Parse(idStr)
	assert.NoError(t, err, "NewString should return a valid UUID string")
	assert.Equal(t, uuid.Version(7), id.Version(), "UUID should be version 7")
	assert.Equal(t, uuid.RFC4122, id.Variant(), "UUID should have RFC4122 variant")

	// Test uniqueness
	idStr2 := NewString()
	assert.NotEqual(t, idStr, idStr2, "Generated UUID strings should be unique")
}

func TestNewString_Format(t *testing.T) {
	// Test that the string format matches UUID format
	idStr := NewString()
	assert.Regexp(t, "^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$", idStr,
		"UUID string should match standard UUID v7 format")
}
