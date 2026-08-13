// Package log provides zerolog logging adapters implementing gtest.Logger and gtest.LogEvent.
package log

import (
	"io"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/rs/zerolog"
)

// Logger is a zerolog-backed implementation of gtest.Logger.
type Logger struct {
	zlog zerolog.Logger
}

// New creates a new Logger writing JSON logs to w at the specified zerolog Level.
func New(w io.Writer, level zerolog.Level) *Logger {
	zlog := zerolog.New(w).Level(level).With().Timestamp().Logger()
	return &Logger{zlog: zlog}
}

// NewWithZerolog wraps an existing zerolog.Logger into a gtest.Logger.
func NewWithZerolog(zlog zerolog.Logger) *Logger {
	return &Logger{zlog: zlog}
}

// WithScenario returns a child Logger with the "scenario" field bound.
func (l *Logger) WithScenario(scenario string) *Logger {
	return &Logger{zlog: l.zlog.With().Str("scenario", scenario).Logger()}
}

// WithVU returns a child Logger with the "vu_id" field bound.
func (l *Logger) WithVU(vuID int) *Logger {
	return &Logger{zlog: l.zlog.With().Int("vu_id", vuID).Logger()}
}

// WithIteration returns a child Logger with the "iteration" field bound.
func (l *Logger) WithIteration(iter int64) *Logger {
	return &Logger{zlog: l.zlog.With().Int64("iteration", iter).Logger()}
}

// WithFields returns a child Logger with arbitrary key-value context fields.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	ctx := l.zlog.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{zlog: ctx.Logger()}
}

// Zerolog returns the underlying zerolog.Logger instance.
func (l *Logger) Zerolog() zerolog.Logger {
	return l.zlog
}

// Debug starts a new debug log event.
func (l *Logger) Debug() gtest.LogEvent {
	return &logEvent{event: l.zlog.Debug()}
}

// Info starts a new info log event.
func (l *Logger) Info() gtest.LogEvent {
	return &logEvent{event: l.zlog.Info()}
}

// Warn starts a new warning log event.
func (l *Logger) Warn() gtest.LogEvent {
	return &logEvent{event: l.zlog.Warn()}
}

// Error starts a new error log event.
func (l *Logger) Error() gtest.LogEvent {
	return &logEvent{event: l.zlog.Error()}
}

// logEvent implements gtest.LogEvent by wrapping a zerolog.Event.
type logEvent struct {
	event *zerolog.Event
}

func (e *logEvent) Str(key, val string) gtest.LogEvent {
	if e.event != nil {
		e.event.Str(key, val)
	}
	return e
}

func (e *logEvent) Int(key string, val int) gtest.LogEvent {
	if e.event != nil {
		e.event.Int(key, val)
	}
	return e
}

func (e *logEvent) Int64(key string, val int64) gtest.LogEvent {
	if e.event != nil {
		e.event.Int64(key, val)
	}
	return e
}

func (e *logEvent) Float64(key string, val float64) gtest.LogEvent {
	if e.event != nil {
		e.event.Float64(key, val)
	}
	return e
}

func (e *logEvent) Bool(key string, val bool) gtest.LogEvent {
	if e.event != nil {
		e.event.Bool(key, val)
	}
	return e
}

func (e *logEvent) Dur(key string, val time.Duration) gtest.LogEvent {
	if e.event != nil {
		e.event.Dur(key, val)
	}
	return e
}

func (e *logEvent) Err(err error) gtest.LogEvent {
	if e.event != nil {
		e.event.Err(err)
	}
	return e
}

func (e *logEvent) Msg(msg string) {
	if e.event != nil {
		e.event.Msg(msg)
	}
}

// Compile-time interface satisfaction checks.
var (
	_ gtest.Logger   = (*Logger)(nil)
	_ gtest.LogEvent = (*logEvent)(nil)
)
