package report

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/sla"
)

// ReportData gathers all scenario metadata, metric store snapshots, and SLA results for reporting.
type ReportData struct {
	SuiteName  string
	Scenario   string
	Version    string
	Commit     string
	StartedAt  time.Time
	EndedAt    time.Time
	Config     config.ScenarioConfig
	Metrics     *metric.Store
	Thresholds  []sla.ThresholdResult
	Passed      bool
	Aborted     bool
	AbortReason string
}

// WriteReport outputs the report in the requested format (console or json) to w or a file.
func WriteReport(w io.Writer, format string, reportOutFile string, data ReportData) error {
	targetWriter := w
	if reportOutFile != "" {
		f, err := os.Create(reportOutFile)
		if err != nil {
			return fmt.Errorf("failed to create report output file %q: %w", reportOutFile, err)
		}
		defer func() {
			_ = f.Close()
		}()
		targetWriter = f
	}


	switch strings.ToLower(format) {
	case "json":
		return GenerateJSONReport(targetWriter, data)
	default:
		return GenerateConsoleReport(targetWriter, data)
	}
}
