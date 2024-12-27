package slogx

import (
	"fmt"
	"log/slog"
)

// Error returns a slog.Attr representing the provided error.
// The attribute key is "error" and the value is the error's message.
//
// Parameters:
//   - err: The error to be converted into a slog.Attr.
//
// Returns:
//   - slog.Attr: An attribute with the key "error" and the error's message as the value.
func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

// ByteString creates a slog.Attr with the given key and a string representation of the byte slice value.
// It converts the byte slice to a string and uses slog.String to create the attribute.
//
// Parameters:
//   - key: The key for the attribute.
//   - value: The byte slice to be converted to a string.
//
// Returns:
//
//	A slog.Attr containing the key and the string representation of the byte slice value.
func ByteString(key string, value []byte) slog.Attr {
	return slog.String(key, string(value))
}

// Stringer creates a slog.Attr with the provided key and the string representation
// of the given fmt.Stringer value. This function is useful for logging purposes
// where you want to include a string representation of an object that implements
// the fmt.Stringer interface.
//
// Parameters:
//   - key: A string representing the key for the attribute.
//   - value: An object that implements the fmt.Stringer interface.
//
// Returns:
//   - slog.Attr: An attribute containing the key and the string representation of the value.
func Stringer(key string, value fmt.Stringer) slog.Attr {
	return slog.String(key, value.String())
}

const (
	// KeyLoggerName is the key for the logger used by Radar.
	KeyLoggerName = "logger"
)

// LoggerName returns an attribute for the logger name.
// LoggerName creates a slog.Attr with the provided logger name.
// The attribute key is defined by KeyLoggerName.
//
// Parameters:
//   - name: The name of the logger.
//
// Returns:
//
//	A slog.Attr containing the logger name.
func LoggerName(name string) slog.Attr {
	return slog.String(KeyLoggerName, name)
}
