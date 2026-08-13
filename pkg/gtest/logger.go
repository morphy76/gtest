package gtest

import "github.com/morphy76/gtest/internal/log"

// LogEvent is a fluent builder for a single log message.
// Callers must terminate with Msg() to emit the event.
type LogEvent = log.LogEvent

// Logger is the scoped logger available inside a VU execution.
// The implementation is zerolog; the interface is stable.
type Logger = log.Logger
