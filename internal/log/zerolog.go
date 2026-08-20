// Package log provides zerolog logging adapters implementing vuhive.Logger and vuhive.LogEvent.
package log

import (
	"io"
	"time"

	"github.com/rs/zerolog"
)

// ZerologLogger is a zerolog-backed implementation of Logger.
type ZerologLogger struct {
	zlog zerolog.Logger
}

// New creates a new ZerologLogger writing JSON logs to w at the specified zerolog Level.
func New(w io.Writer, level zerolog.Level) *ZerologLogger {
	writer := zerolog.SyncWriter(w)
	zlog := zerolog.New(writer).Level(level).With().Timestamp().Logger()
	return &ZerologLogger{zlog: zlog}
}

// NewWithFormat creates a ZerologLogger that uses either human-readable console output ("pretty")
// or structured JSON output ("json"). Defaults to JSON for unrecognized formats.
func NewWithFormat(w io.Writer, level zerolog.Level, format string) *ZerologLogger {
	var writer io.Writer = w
	if format == "pretty" {
		writer = zerolog.ConsoleWriter{Out: w, TimeFormat: time.RFC3339}
	}
	writer = zerolog.SyncWriter(writer)
	zlog := zerolog.New(writer).Level(level).With().Timestamp().Logger()

	return &ZerologLogger{zlog: zlog}
}

// NewWithZerolog wraps an existing zerolog.Logger into a ZerologLogger.
func NewWithZerolog(zlog zerolog.Logger) *ZerologLogger {
	return &ZerologLogger{zlog: zlog}
}

// WithScenario returns a child ZerologLogger with the "scenario" field bound.
func (l *ZerologLogger) WithScenario(scenario string) *ZerologLogger {
	return &ZerologLogger{zlog: l.zlog.With().Str("scenario", scenario).Logger()}
}

// WithVU returns a child ZerologLogger with the "vu_id" field bound.
func (l *ZerologLogger) WithVU(vuID int) *ZerologLogger {
	return &ZerologLogger{zlog: l.zlog.With().Int("vu_id", vuID).Logger()}
}

// WithIteration returns a child ZerologLogger with the "iteration" field bound.
func (l *ZerologLogger) WithIteration(iter int64) *ZerologLogger {
	return &ZerologLogger{zlog: l.zlog.With().Int64("iteration", iter).Logger()}
}

// WithFields returns a child ZerologLogger with arbitrary key-value context fields.
func (l *ZerologLogger) WithFields(fields map[string]any) *ZerologLogger {
	ctx := l.zlog.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &ZerologLogger{zlog: ctx.Logger()}
}

// Zerolog returns the underlying zerolog.Logger instance.
func (l *ZerologLogger) Zerolog() zerolog.Logger {
	return l.zlog
}

// Debug starts a new debug log event.
func (l *ZerologLogger) Debug() LogEvent {
	return &logEvent{event: l.zlog.Debug()}
}

// Info starts a new info log event.
func (l *ZerologLogger) Info() LogEvent {
	return &logEvent{event: l.zlog.Info()}
}

// Warn starts a new warning log event.
func (l *ZerologLogger) Warn() LogEvent {
	return &logEvent{event: l.zlog.Warn()}
}

// Error starts a new error log event.
func (l *ZerologLogger) Error() LogEvent {
	return &logEvent{event: l.zlog.Error()}
}

// logEvent implements vuhive.LogEvent by wrapping a zerolog.Event.
type logEvent struct {
	event *zerolog.Event
}

func (e *logEvent) Str(key, val string) LogEvent {
	if e.event != nil {
		e.event.Str(key, val)
	}
	return e
}

func (e *logEvent) Int(key string, val int) LogEvent {
	if e.event != nil {
		e.event.Int(key, val)
	}
	return e
}

func (e *logEvent) Int64(key string, val int64) LogEvent {
	if e.event != nil {
		e.event.Int64(key, val)
	}
	return e
}

func (e *logEvent) Float64(key string, val float64) LogEvent {
	if e.event != nil {
		e.event.Float64(key, val)
	}
	return e
}

func (e *logEvent) Bool(key string, val bool) LogEvent {
	if e.event != nil {
		e.event.Bool(key, val)
	}
	return e
}

func (e *logEvent) Dur(key string, val time.Duration) LogEvent {
	if e.event != nil {
		e.event.Dur(key, val)
	}
	return e
}

func (e *logEvent) Err(err error) LogEvent {
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
	_ Logger   = (*ZerologLogger)(nil)
	_ LogEvent = (*logEvent)(nil)
)
