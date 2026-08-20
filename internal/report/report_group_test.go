package report_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/config"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/morphy76/vuhive/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateConsoleReport_WithGroups(t *testing.T) {
	store := metric.NewStore()
	h1 := store.Duration(metric.GroupMetricName("01_Login"), nil)
	h1.Observe(15 * time.Millisecond)

	h2 := store.Duration(metric.GroupMetricName("03_Checkout::Payment"), nil)
	h2.Observe(45 * time.Millisecond)

	var buf bytes.Buffer
	data := report.ReportData{
		Scenario:  "group_demo",
		Version:   "1.0.0",
		Commit:    "abcdef",
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(1 * time.Second),
		Config: config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       2,
			RunPeriod: 1 * time.Second,
		},
		Metrics: store,
		Passed:  true,
	}

	err := report.GenerateConsoleReport(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "GROUPS")
	assert.Contains(t, out, "01_Login")
	assert.Contains(t, out, "03_Checkout::Payment")
	assert.Contains(t, out, "Group Name")
}

func TestGenerateConsoleReport_WithoutGroups(t *testing.T) {
	store := metric.NewStore()
	var buf bytes.Buffer
	data := report.ReportData{
		Scenario:  "no_group_demo",
		Version:   "1.0.0",
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(1 * time.Second),
		Config: config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       2,
			RunPeriod: 1 * time.Second,
		},
		Metrics: store,
		Passed:  true,
	}

	err := report.GenerateConsoleReport(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.NotContains(t, out, "GROUPS")
}

func TestGenerateJSONReport_WithGroups(t *testing.T) {
	store := metric.NewStore()
	h1 := store.Duration(metric.GroupMetricName("01_Login"), nil)
	h1.Observe(20 * time.Millisecond)

	h2 := store.Duration(metric.GroupMetricName("02_Checkout"), nil)
	h2.Observe(80 * time.Millisecond)

	var buf bytes.Buffer
	data := report.ReportData{
		Scenario:  "json_group_demo",
		Version:   "1.0.0",
		Commit:    "123456",
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(1 * time.Second),
		Config: config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       1,
			RunPeriod: 1 * time.Second,
		},
		Metrics: store,
		Passed:  true,
	}

	err := report.GenerateJSONReport(&buf, data)
	require.NoError(t, err)

	var doc struct {
		Groups []struct {
			Name   string `json:"name"`
			Count  int64  `json:"count"`
			MinMS  int64  `json:"min_ms"`
			MaxMS  int64  `json:"max_ms"`
			P95MS  int64  `json:"p95_ms"`
			MeanMS int64  `json:"mean_ms"`
		} `json:"groups"`
	}

	err = json.Unmarshal(buf.Bytes(), &doc)
	require.NoError(t, err)
	require.Len(t, doc.Groups, 2)
	assert.Equal(t, "01_Login", doc.Groups[0].Name)
	assert.Equal(t, int64(1), doc.Groups[0].Count)
	assert.Equal(t, "02_Checkout", doc.Groups[1].Name)
	assert.Equal(t, int64(1), doc.Groups[1].Count)
}

func TestSummaryData_GroupHelper(t *testing.T) {
	summary := report.SummaryData{
		Groups: []report.GroupSummary{
			{Name: "login", Count: 10, Mean: 15 * time.Millisecond},
			{Name: "checkout::pay", Count: 5, Mean: 50 * time.Millisecond},
		},
	}

	grp := summary.Group("login")
	require.NotNil(t, grp)
	assert.Equal(t, int64(10), grp.Count)

	nestedGrp := summary.Group("checkout::pay")
	require.NotNil(t, nestedGrp)
	assert.Equal(t, int64(5), nestedGrp.Count)

	assert.Nil(t, summary.Group("nonexistent"))
}
