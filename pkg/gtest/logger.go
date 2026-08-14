package gtest

import "time"

// LogEvent is a fluent builder for a single log message.
// Callers must terminate with Msg() to emit the event.
type LogEvent interface {
	// Str appends a string key-value pair to the log event.
	Str(key, val string) LogEvent

	// Int appends an int key-value pair to the log event.
	Int(key string, val int) LogEvent

	// Int64 appends an int64 key-value pair to the log event.
	Int64(key string, val int64) LogEvent

	// Float64 appends a float64 key-value pair to the log event.
	Float64(key string, val float64) LogEvent

	// Bool appends a boolean key-value pair to the log event.
	Bool(key string, val bool) LogEvent

	// Dur appends a time.Duration key-value pair to the log event.
	Dur(key string, val time.Duration) LogEvent

	// Err appends an error under the "error" key to the log event.
	Err(err error) LogEvent

	// Msg emits the structured log event with the specified message text.
	Msg(msg string)
}

// Logger is the scoped logger available inside a VU execution.
// The underlying implementation is Zerolog; the interface provides structured logging methods.
type Logger interface {
	// Debug starts a new debug-level log event.
	Debug() LogEvent

	// Info starts a new info-level log event.
	Info() LogEvent

	// Warn starts a new warning-level log event.
	Warn() LogEvent

	// Error starts a new error-level log event.
	Error() LogEvent
}
