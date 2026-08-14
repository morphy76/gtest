package report

import (
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/metric"
)

type jsonMetricEntry struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Tags   map[string]string `json:"tags,omitempty"`
	Count  int64             `json:"count,omitempty"`
	Value  float64           `json:"value,omitempty"`
	Rate   float64           `json:"rate,omitempty"`
	MinMS  int64             `json:"min_ms,omitempty"`
	MeanMS int64             `json:"mean_ms,omitempty"`
	P50MS  int64             `json:"p50_ms,omitempty"`
	P90MS  int64             `json:"p90_ms,omitempty"`
	P95MS  int64             `json:"p95_ms,omitempty"`
	P99MS  int64             `json:"p99_ms,omitempty"`
	MaxMS  int64             `json:"max_ms,omitempty"`
}

type jsonThresholdEntry struct {
	Metric   string `json:"metric"`
	Stat     string `json:"stat"`
	Operator string `json:"operator"`
	Target   string `json:"target"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

type jsonReportDocument struct {
	SuiteName   string                `json:"suite_name"`
	Scenario    string                `json:"scenario"`
	Version     string                `json:"version"`
	Commit      string                `json:"commit"`
	StartedAt   time.Time             `json:"started_at"`
	EndedAt     time.Time             `json:"ended_at"`
	Config      config.ScenarioConfig `json:"config"`
	Metrics     []jsonMetricEntry     `json:"metrics"`
	Checks      []metric.CheckSummary `json:"checks,omitempty"`
	Thresholds  []jsonThresholdEntry  `json:"thresholds"`
	Passed      bool                  `json:"passed"`
	Aborted     bool                  `json:"aborted"`
	AbortReason string                `json:"abort_reason,omitempty"`
}

// GenerateJSONReport formats and writes the JSON report document to w.
func GenerateJSONReport(w io.Writer, data ReportData) error {
	var checks []metric.CheckSummary
	if data.Metrics != nil {
		checks = data.Metrics.CheckSummaries()
	}

	doc := jsonReportDocument{
		SuiteName:   sOrDefault(data.SuiteName, "gtest"),
		Scenario:    data.Scenario,
		Version:     data.Version,
		Commit:      data.Commit,
		StartedAt:   data.StartedAt.UTC(),
		EndedAt:     data.EndedAt.UTC(),
		Config:      data.Config,
		Checks:      checks,
		Passed:      data.Passed,
		Aborted:     data.Aborted,
		AbortReason: data.AbortReason,
	}

	// Format threshold entries
	for _, th := range data.Thresholds {
		doc.Thresholds = append(doc.Thresholds, jsonThresholdEntry{
			Metric:   th.Metric,
			Stat:     th.Stat,
			Operator: th.Operator,
			Target:   th.Target,
			Actual:   th.Actual,
			Passed:   th.Passed,
		})
	}

	// Format metrics entries sorted alphabetically by name
	type namedEntry struct {
		name string
		item jsonMetricEntry
	}
	var entries []namedEntry

	// Histograms
	for _, name := range data.Metrics.HistogramNames() {
		snap := data.Metrics.MergedHistogramSnapshot(name)
		entries = append(entries, namedEntry{
			name: name,
			item: jsonMetricEntry{
				Name:   name,
				Type:   "duration",
				Count:  snap.Count,
				MinMS:  snap.Min.Milliseconds(),
				MeanMS: snap.Mean.Milliseconds(),
				P50MS:  snap.P50.Milliseconds(),
				P90MS:  snap.P90.Milliseconds(),
				P95MS:  snap.P95.Milliseconds(),
				P99MS:  snap.P99.Milliseconds(),
				MaxMS:  snap.Max.Milliseconds(),
			},
		})
	}

	// Counters
	for _, name := range data.Metrics.CounterNames() {
		val := data.Metrics.AggregatedCounterValue(name)
		entries = append(entries, namedEntry{
			name: name,
			item: jsonMetricEntry{
				Name:  name,
				Type:  "counter",
				Count: val,
			},
		})
	}

	// Gauges
	for _, name := range data.Metrics.GaugeNames() {
		val := data.Metrics.LastGaugeValue(name)
		entries = append(entries, namedEntry{
			name: name,
			item: jsonMetricEntry{
				Name:  name,
				Type:  "gauge",
				Value: val,
			},
		})
	}

	// Rates
	for _, name := range data.Metrics.RateNames() {
		val := data.Metrics.AggregatedRateValue(name)
		entries = append(entries, namedEntry{
			name: name,
			item: jsonMetricEntry{
				Name: name,
				Type: "rate",
				Rate: val,
			},
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	for _, e := range entries {
		doc.Metrics = append(doc.Metrics, e.item)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func sOrDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
