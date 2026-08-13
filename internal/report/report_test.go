package report_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/report"
	"github.com/morphy76/gtest/internal/sla"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestReportData() report.ReportData {
	store := metric.NewStore()

	// Built-in metrics
	store.Counter("gtest.vu.iterations_total", metric.Tags{}).Add(100)
	store.Counter("gtest.vu.iterations_failed", metric.Tags{}).Add(2)
	store.Counter("gtest.vu.iterations_timeout", metric.Tags{}).Add(1)

	// Custom metrics (out of order to test alphabetical sorting)
	store.Counter("z_requests_total", metric.Tags{}).Add(100)
	store.Duration("http_request_duration", metric.Tags{}).Observe(50 * time.Millisecond)
	store.Rate("alpha_success_rate", metric.Tags{}).Add(98, 100)
	store.Gauge("beta_gauge", metric.Tags{}).Set(42.5)

	thresholds := []sla.ThresholdResult{
		{
			Metric:   "http_request_duration",
			Stat:     "p95",
			Operator: "<",
			Target:   "200ms",
			Actual:   "50ms",
			Passed:   true,
		},
		{
			Metric:   "alpha_success_rate",
			Stat:     "rate",
			Operator: ">=",
			Target:   "0.99",
			Actual:   "0.98",
			Passed:   false,
			Reason:   "actual 0.98 < target 0.99",
		},
	}

	startedAt := time.Now().Add(-10 * time.Second)
	endedAt := time.Now()

	return report.ReportData{
		SuiteName: "Test Suite",
		Scenario:  "http_checkout_flow",
		Version:   "0.1.0",
		Commit:    "a1b2c3d",
		StartedAt: startedAt,
		EndedAt:   endedAt,
		Config: config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       10,
			RampUp:    2 * time.Second,
			RunPeriod: 5 * time.Second,
			RampDown:  1 * time.Second,
			VUTimeout: 1 * time.Second,
		},
		Metrics:    store,
		Thresholds: thresholds,
		Passed:     false,
	}
}

// AC-1.9.1: Console report contains scenario name, mode, duration, iteration counts
func TestConsoleReportContainsMetadataAndCounts(t *testing.T) {
	data := createTestReportData()
	var buf bytes.Buffer

	err := report.GenerateConsoleReport(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Scenario:     http_checkout_flow")
	assert.Contains(t, out, "Mode:         constant_vus (10 VUs)")
	assert.Contains(t, out, "Duration:")
	assert.Contains(t, out, "100 total")
	assert.Contains(t, out, "2 failed")
	assert.Contains(t, out, "1 timeout")
}

// AC-1.9.2: Console report shows [PASS]/[FAIL] for each threshold with actual vs target value
func TestConsoleReportShowsThresholdPassFail(t *testing.T) {
	data := createTestReportData()
	var buf bytes.Buffer

	err := report.GenerateConsoleReport(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "[PASS]  http_request_duration   p95 < 200ms     → actual: 50ms")
	assert.Contains(t, out, "[FAIL]  alpha_success_rate      rate >= 0.99    → actual: 0.98")
	assert.Contains(t, out, "OVERALL: FAILED")
}

// AC-1.9.3: JSON report is valid JSON and unmarshals to the schema defined in §10.2
func TestJSONReportValidAndMatchesSchema(t *testing.T) {
	data := createTestReportData()
	var buf bytes.Buffer

	err := report.GenerateJSONReport(&buf, data)
	require.NoError(t, err)

	var doc struct {
		SuiteName  string `json:"suite_name"`
		Scenario   string `json:"scenario"`
		Version    string `json:"version"`
		Commit     string `json:"commit"`
		StartedAt  string `json:"started_at"`
		EndedAt    string `json:"ended_at"`
		Config     any    `json:"config"`
		Metrics    []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"metrics"`
		Thresholds []struct {
			Metric   string `json:"metric"`
			Stat     string `json:"stat"`
			Operator string `json:"operator"`
			Target   string `json:"target"`
			Actual   string `json:"actual"`
			Passed   bool   `json:"passed"`
		} `json:"thresholds"`
		Passed bool `json:"passed"`
	}

	err = json.Unmarshal(buf.Bytes(), &doc)
	require.NoError(t, err, "output must be valid JSON matching §10.2 schema")

	assert.Equal(t, "Test Suite", doc.SuiteName)
	assert.Equal(t, "http_checkout_flow", doc.Scenario)
	assert.Equal(t, "0.1.0", doc.Version)
	assert.Equal(t, "a1b2c3d", doc.Commit)
	assert.False(t, doc.Passed)
	assert.Len(t, doc.Thresholds, 2)
}

// AC-1.9.4: Metrics in both report formats are sorted alphabetically by name
func TestMetricsSortedAlphabeticallyInBothFormats(t *testing.T) {
	data := createTestReportData()

	// Console report test
	var consoleBuf bytes.Buffer
	require.NoError(t, report.GenerateConsoleReport(&consoleBuf, data))
	consoleOut := consoleBuf.String()

	idxAlpha := strings.Index(consoleOut, "alpha_success_rate")
	idxBeta := strings.Index(consoleOut, "beta_gauge")
	idxHTTP := strings.Index(consoleOut, "http_request_duration")
	idxZ := strings.Index(consoleOut, "z_requests_total")

	require.True(t, idxAlpha >= 0 && idxBeta >= 0 && idxHTTP >= 0 && idxZ >= 0, "all metrics must be present in console report")
	assert.True(t, idxAlpha < idxBeta, "alpha_success_rate must come before beta_gauge")
	assert.True(t, idxBeta < idxHTTP, "beta_gauge must come before http_request_duration")
	assert.True(t, idxHTTP < idxZ, "http_request_duration must come before z_requests_total")

	// JSON report test
	var jsonBuf bytes.Buffer
	require.NoError(t, report.GenerateJSONReport(&jsonBuf, data))

	var doc struct {
		Metrics []struct {
			Name string `json:"name"`
		} `json:"metrics"`
	}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &doc))

	var names []string
	for _, m := range doc.Metrics {
		names = append(names, m.Name)
	}

	require.Contains(t, names, "alpha_success_rate")
	require.Contains(t, names, "beta_gauge")
	require.Contains(t, names, "http_request_duration")
	require.Contains(t, names, "z_requests_total")

	for i := 0; i < len(names)-1; i++ {
		assert.True(t, names[i] <= names[i+1], "JSON metrics must be sorted alphabetically: %s <= %s", names[i], names[i+1])
	}
}

// AC-1.9.5: --report-out writes to the specified file, not stdout
func TestWriteReportToFile(t *testing.T) {
	data := createTestReportData()
	tempDir := t.TempDir()
	outFilePath := filepath.Join(tempDir, "custom_report.txt")

	var stdoutBuf bytes.Buffer
	err := report.WriteReport(&stdoutBuf, "console", outFilePath, data)
	require.NoError(t, err)

	assert.Empty(t, stdoutBuf.String(), "stdout buffer must remain empty when reportOut is specified")

	fileBytes, err := os.ReadFile(outFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(fileBytes), "GTEST LOAD TEST SUMMARY")
}
