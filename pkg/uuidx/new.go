package uuidx

import "github.com/google/uuid"

// New generates a new UUID using the version 7 format and returns it.
// It panics if the UUID generation fails.
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// NewString generates a new UUID using the version 7 format and returns it as a string.
// It utilizes the New function to create the UUID and then converts it to a string.
func NewString() string {
	return New().String()
}
