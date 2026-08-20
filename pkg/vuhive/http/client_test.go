package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/morphy76/vuhive/pkg/vuhive"
	vuhivehttp "github.com/morphy76/vuhive/pkg/vuhive/http"
)

// metricsAdapter wraps internal metric.Collector to satisfy vuhive.MetricsCollector.
type metricsAdapter struct {
	collector metric.Collector
}

func (m *metricsAdapter) Counter(name string, tags vuhive.Tags) vuhive.Counter {
	return m.collector.Counter(name, metric.Tags(tags))
}

func (m *metricsAdapter) Gauge(name string, tags vuhive.Tags) vuhive.Gauge {
	return m.collector.Gauge(name, metric.Tags(tags))
}

func (m *metricsAdapter) Duration(name string, tags vuhive.Tags) vuhive.Duration {
	return m.collector.Duration(name, metric.Tags(tags))
}

func (m *metricsAdapter) Rate(name string, tags vuhive.Tags) vuhive.Rate {
	return m.collector.Rate(name, metric.Tags(tags))
}

// newTestStore creates a metric store and a MetricsCollector adapter for testing.
func newTestStore(t *testing.T) (*metric.Store, vuhive.MetricsCollector) {
	t.Helper()
	store := metric.NewStore()
	adapter := &metricsAdapter{collector: store}
	return store, adapter
}

// newTestServer creates a simple httptest.Server that returns JSON.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Echo-Method", r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
}

// newErrorServer creates a server that returns 500.
func newErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
}

// --- Client constructor tests ---

func TestNewClient_DefaultOptions(t *testing.T) {
	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)
	assert.NotNil(t, client, "client should not be nil")
}

func TestNewClient_WithTimeout(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithTimeout(5*time.Second))

	resp, err := client.Get(context.Background(), ts.URL+"/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewClient_WithHeaders(t *testing.T) {
	var capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics,
		vuhivehttp.WithHeader("Authorization", "Bearer test-token"),
	)

	_, err := client.Get(context.Background(), ts.URL+"/test")
	require.NoError(t, err)
	assert.Equal(t, "Bearer test-token", capturedAuth)
}

func TestNewClient_WithBulkHeaders(t *testing.T) {
	var capturedAuth, capturedAccept string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics,
		vuhivehttp.WithHeaders(map[string]string{
			"Authorization": "Bearer bulk-token",
			"Accept":        "application/json",
		}),
	)

	_, err := client.Get(context.Background(), ts.URL+"/test")
	require.NoError(t, err)
	assert.Equal(t, "Bearer bulk-token", capturedAuth)
	assert.Equal(t, "application/json", capturedAccept)
}

func TestNewClient_WithHeaders_DoNotOverwriteExplicit(t *testing.T) {
	var capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics,
		vuhivehttp.WithHeader("Authorization", "Bearer default"),
	)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/test", nil)
	req.Header.Set("Authorization", "Bearer explicit")
	_, err := client.Do(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Bearer explicit", capturedAuth, "explicit header should not be overwritten")
}

func TestNewClient_WithCustomMetricPrefix(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics,
		vuhivehttp.WithCustomMetricPrefix("custom."),
	)

	_, err := client.Get(context.Background(), ts.URL+"/test")
	require.NoError(t, err)

	counterVal := store.AggregatedCounterValue("custom.reqs")
	assert.Equal(t, int64(1), counterVal, "custom prefix counter should be incremented")
}

func TestNewClient_WithTLSInsecureSkipVerify(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)

	// Without InsecureSkipVerify, HTTPS to a test server with self-signed cert should fail
	clientStrict := vuhivehttp.NewClientWithCollector(metrics)
	_, err := clientStrict.Get(context.Background(), ts.URL+"/test")
	assert.Error(t, err, "strict TLS should reject self-signed cert")

	// With InsecureSkipVerify, it should succeed
	clientInsecure := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithTLSInsecureSkipVerify())
	resp, err := clientInsecure.Get(context.Background(), ts.URL+"/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- Request method tests ---

func TestClient_Get_RecordsMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	resp, err := client.Get(context.Background(), ts.URL+"/api/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify request counter
	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(1), counterVal, "request counter should be 1")

	// Verify duration was recorded
	snap := store.MergedHistogramSnapshot(vuhive.MetricHTTPReqDuration)
	assert.Equal(t, int64(1), snap.Count, "duration should have 1 observation")
}

func TestClient_Post_RecordsMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	body := strings.NewReader(`{"item":"test"}`)
	resp, err := client.Post(context.Background(), ts.URL+"/api/items", "application/json", body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(1), counterVal)
}

func TestClient_Put_RecordsMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	body := strings.NewReader(`{"item":"updated"}`)
	resp, err := client.Put(context.Background(), ts.URL+"/api/items/1", "application/json", body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(1), counterVal)
}

func TestClient_Delete_RecordsMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	resp, err := client.Delete(context.Background(), ts.URL+"/api/items/1")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(1), counterVal)
}

func TestClient_Do_RecordsMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, ts.URL+"/api/items/1", nil)
	require.NoError(t, err)

	resp, err := client.Do(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(1), counterVal)
}

func TestClient_Get_NonOKStatus_RecordsFailedRate(t *testing.T) {
	ts := newErrorServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	resp, err := client.Get(context.Background(), ts.URL+"/api/fail")
	require.NoError(t, err, "non-OK status is not a transport error")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Verify failure rate was recorded (1.0 = 100% failures)
	failedRate := store.AggregatedRateValue(vuhive.MetricHTTPReqFailed)
	assert.Equal(t, 1.0, failedRate, "failure rate should be 1.0 for failed request")
}

func TestClient_Get_SuccessStatus_RecordsSuccessRate(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	_, err := client.Get(context.Background(), ts.URL+"/api/test")
	require.NoError(t, err)

	// Verify success rate (0.0 = 0% failures)
	failedRate := store.AggregatedRateValue(vuhive.MetricHTTPReqFailed)
	assert.Equal(t, 0.0, failedRate, "failure rate should be 0.0 for successful request")
}

func TestClient_Get_FailedRequest_RecordsFailedRate(t *testing.T) {
	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithTimeout(100*time.Millisecond))

	// Request to a non-existent address
	_, err := client.Get(context.Background(), "http://127.0.0.1:1/nonexistent")
	assert.Error(t, err)

	// Metrics should still be recorded for the failed request
	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(1), counterVal, "counter should be 1 even for failed request")

	failedRate := store.AggregatedRateValue(vuhive.MetricHTTPReqFailed)
	assert.Equal(t, 1.0, failedRate, "failure rate should be 1.0 for transport failure")
}

func TestClient_Get_RespectsContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithTimeout(1*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, ts.URL+"/slow")
	assert.Error(t, err, "cancelled context should return an error")
}

func TestClient_Get_ResponseBodyParsing(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	resp, err := client.Get(context.Background(), ts.URL+"/api/test")
	require.NoError(t, err)

	// Verify Text()
	assert.Equal(t, `{"status":"ok"}`, resp.Text())

	// Verify JSON()
	var result map[string]string
	require.NoError(t, resp.JSON(&result))
	assert.Equal(t, "ok", result["status"])

	// Verify Headers
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
}

func TestClient_MultipleRequests_AccumulateMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	for i := 0; i < 5; i++ {
		_, err := client.Get(context.Background(), ts.URL+"/api/test")
		require.NoError(t, err)
	}

	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(5), counterVal, "5 requests should produce 5 counter increments")
}

// --- Phase timing tests (opt-in) ---

func TestClient_Get_DetailedTiming_Disabled_NoPhaseMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics) // No WithDetailedTiming

	_, err := client.Get(context.Background(), ts.URL+"/api/test")
	require.NoError(t, err)

	// Phase metrics should NOT be recorded
	snap := store.MergedHistogramSnapshot(vuhive.MetricHTTPReqConnecting)
	assert.Equal(t, int64(0), snap.Count, "connecting metric should not be recorded without detailed timing")
}

func TestClient_Get_DetailedTiming_Enabled_RecordsPhaseMetrics(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithDetailedTiming())

	resp, err := client.Get(context.Background(), ts.URL+"/api/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Core metrics should always be recorded
	counterVal := store.AggregatedCounterValue(vuhive.MetricHTTPReqs)
	assert.Equal(t, int64(1), counterVal)

	// Sending and receiving should be recorded for a successful request
	sendSnap := store.MergedHistogramSnapshot(vuhive.MetricHTTPReqSending)
	recvSnap := store.MergedHistogramSnapshot(vuhive.MetricHTTPReqReceiving)
	assert.Equal(t, int64(1), sendSnap.Count, "sending duration should be recorded with detailed timing")
	assert.Equal(t, int64(1), recvSnap.Count, "receiving duration should be recorded with detailed timing")
}
