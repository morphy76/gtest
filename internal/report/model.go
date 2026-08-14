package report

import (
	"io"
	"strings"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/sla"
)

// ReportData gathers all scenario metadata, metric store snapshots, and SLA results for reporting.
type ReportData struct {
	SuiteName   string
	Scenario    string
	Version     string
	Commit      string
	StartedAt   time.Time
	EndedAt     time.Time
	Config      config.ScenarioConfig
	Metrics     metric.Reader
	Thresholds  []sla.ThresholdResult
	Passed      bool
	Aborted     bool
	AbortReason string
}

// WriteReport outputs the report in the requested format (console or json) to w.
func WriteReport(w io.Writer, format string, data ReportData) error {
	switch strings.ToLower(format) {
	case "json":
		return GenerateJSONReport(w, data)
	default:
		return GenerateConsoleReport(w, data)
	}
}

