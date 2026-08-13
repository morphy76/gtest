package gtest

import "time"

// LogEvent is a fluent builder for a single log message.
// Callers must terminate with Msg() to emit the event.
type LogEvent interface {
	Str(key, val string) LogEvent
	Int(key string, val int) LogEvent
	Int64(key string, val int64) LogEvent
	Float64(key string, val float64) LogEvent
	Bool(key string, val bool) LogEvent
	Dur(key string, val time.Duration) LogEvent
	Err(err error) LogEvent
	Msg(msg string)
}

// Logger is the scoped logger available inside a VU execution.
// The implementation is zerolog; the interface is stable.
type Logger interface {
	Debug() LogEvent
	Info() LogEvent
	Warn() LogEvent
	Error() LogEvent
}
