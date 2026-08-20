package log_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// AC-1.5.1: Log().Info().Str("k","v").Msg("m") emits one JSON line with level=info, k=v, message=m
func TestInfoLogEmitsJSONLine(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, zerolog.InfoLevel)

	logger.Info().Str("k", "v").Msg("m")

	var result map[string]any
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "log output must be valid JSON: %s", buf.String())

	assert.Equal(t, "info", result["level"])
	assert.Equal(t, "v", result["k"])
	assert.Equal(t, "m", result["message"])
}

// AC-1.5.2: Logger built with vuID=3 auto-injects vu_id=3 on every event
func TestAutoInjectsVUID(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := log.New(&buf, zerolog.InfoLevel)
	vuLogger := baseLogger.WithVU(3)

	vuLogger.Info().Str("action", "checkout").Msg("started iteration")

	var result map[string]any
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "log output must be valid JSON: %s", buf.String())

	assert.Equal(t, float64(3), result["vu_id"])
	assert.Equal(t, "checkout", result["action"])
	assert.Equal(t, "started iteration", result["message"])
}

// AC-1.5.3: Debug events are suppressed when log level is set to "info"
func TestDebugEventsSuppressedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, zerolog.InfoLevel)

	// Debug should do nothing and write nothing to buffer.
	logger.Debug().Str("secret", "value").Int("count", 42).Msg("this should be suppressed")
	assert.Empty(t, buf.String(), "buffer must be empty when log event is suppressed")

	// Info should emit.
	logger.Info().Msg("this should emit")
	assert.NotEmpty(t, buf.String(), "buffer must contain output for Info event")
}

// AC-1.5.4: ZerologLogger satisfies the log.Logger interface (compile-time check)
func TestLoggerInterfaceSatisfaction(t *testing.T) {
	l := log.New(io.Discard, zerolog.DebugLevel)
	e := l.Debug()
	assert.NotNil(t, l)
	assert.NotNil(t, e)
}




// Additional test: WithScenario and WithIteration binding
func TestScenarioAndIterationBinding(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, zerolog.DebugLevel).
		WithScenario("checkout_flow").
		WithVU(5).
		WithIteration(42)

	logger.Info().Msg("executing VU")

	var result map[string]any
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "checkout_flow", result["scenario"])
	assert.Equal(t, float64(5), result["vu_id"])
	assert.Equal(t, float64(42), result["iteration"])
}

// Additional test: All LogEvent data types
func TestLogEventDataTypes(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, zerolog.DebugLevel)

	dummyErr := errors.New("network timeout")
	logger.Warn().
		Str("str_key", "hello").
		Int("int_key", 10).
		Int64("int64_key", 10000000000).
		Float64("float_key", 3.14).
		Bool("bool_key", true).
		Dur("dur_key", 200*time.Millisecond).
		Err(dummyErr).
		Msg("warning test")

	var result map[string]any
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "warn", result["level"])
	assert.Equal(t, "hello", result["str_key"])
	assert.Equal(t, float64(10), result["int_key"])
	assert.Equal(t, float64(10000000000), result["int64_key"])
	assert.Equal(t, 3.14, result["float_key"])
	assert.Equal(t, true, result["bool_key"])
	assert.Equal(t, float64(200), result["dur_key"]) // zerolog formats duration as ms float64 by default
	assert.Equal(t, "network timeout", result["error"])
	assert.Equal(t, "warning test", result["message"])
}

// Additional test: Error log level
func TestErrorLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, zerolog.InfoLevel)

	logger.Error().Err(errors.New("db crash")).Msg("failed to process")

	var result map[string]any
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "error", result["level"])
	assert.Equal(t, "db crash", result["error"])
	assert.Equal(t, "failed to process", result["message"])
}

func TestConcurrentLogging_Async(t *testing.T) {
	var buf bytes.Buffer
	logger := log.NewAsync(&buf, zerolog.DebugLevel)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.WithVU(id).WithIteration(int64(id * 2)).Info().Str("key", "val").Msg("test concurrent message")
		}(i)
	}
	wg.Wait()
	require.NoError(t, logger.Close())
	assert.NotEmpty(t, buf.String())
}

func TestConcurrentLogging_AsyncWithFormat_Pretty(t *testing.T) {
	var buf bytes.Buffer
	logger := log.NewAsyncWithFormat(&buf, zerolog.DebugLevel, "pretty")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.WithVU(id).WithIteration(int64(id * 2)).Info().Str("key", "val").Msg("test concurrent pretty message")
		}(i)
	}
	wg.Wait()
	require.NoError(t, logger.Close())
	assert.NotEmpty(t, buf.String())
}
