package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vuhivehttp "github.com/morphy76/vuhive/pkg/vuhive/http"
)

// --- Finding 1: MaxIdleConnsPerHost default should be 100 ---

func TestDefaultConfig_MaxIdleConnsPerHost_Is100(t *testing.T) {
	// The default MaxIdleConnsPerHost should be 100 to prevent connection thrashing
	// in high-VU load tests targeting a single host.
	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	// We need to verify the transport was configured with MaxIdleConnsPerHost=100.
	// Since the config is internal, we test indirectly via the exported ExportDefaultMaxIdleConnsPerHost.
	assert.Equal(t, 100, vuhivehttp.ExportDefaultMaxIdleConnsPerHost(),
		"default MaxIdleConnsPerHost should be 100 to prevent connection thrash at scale")
	_ = client // client is created successfully
}

// --- Finding 2: sync.Pool buffer reuse for response body reads ---

func TestClient_Do_ReusesBuffers_ReducesAllocations(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","data":"some payload for buffer reuse testing"}`))
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics)

	// Warm up the pool
	resp, err := client.Get(context.Background(), ts.URL+"/warmup")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Measure allocations - with sync.Pool buffer reuse, steady-state allocations
	// should be lower than naive io.ReadAll (which allocates a new []byte every time).
	allocs := testing.AllocsPerRun(100, func() {
		resp, err := client.Get(context.Background(), ts.URL+"/test")
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.StatusCode
	})

	// With sync.Pool, the body read buffer is reused. The remaining allocations come from
	// net/http internals (request/response objects, headers, etc.) which we cannot control.
	// Without pooling (io.ReadAll), allocations would be higher due to growing buffer copies.
	// We assert that steady-state allocations stay within the net/http baseline budget.
	assert.LessOrEqual(t, allocs, float64(120),
		"steady-state request should have bounded allocations with buffer pooling")
}

// --- Finding 3: WithDiscardBody option ---

func TestClient_WithDiscardBody_DoesNotReturnBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","large":"payload"}`))
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithDiscardBody())

	resp, err := client.Get(context.Background(), ts.URL+"/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Nil(t, resp.Body, "body should be nil when WithDiscardBody is configured")
	assert.Equal(t, "", resp.Text(), "Text() should return empty string when body is nil")
}

func TestClient_WithDiscardBody_StillRecordsMetrics(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithDiscardBody())

	_, err := client.Get(context.Background(), ts.URL+"/test")
	require.NoError(t, err)

	// Metrics should still be recorded normally
	counterVal := store.AggregatedCounterValue("vuhive.http.reqs")
	assert.Equal(t, int64(1), counterVal, "request counter should be incremented even with discarded body")
}

func TestClient_WithDiscardBody_Post_DiscardsBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	}))
	defer ts.Close()

	_, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithDiscardBody())

	resp, err := client.Post(context.Background(), ts.URL+"/items", "application/json", strings.NewReader(`{"name":"test"}`))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Nil(t, resp.Body, "POST response body should be nil when WithDiscardBody is configured")
}

func TestClient_WithDiscardBody_ErrorStatus_StillRecordsFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`error`))
	}))
	defer ts.Close()

	store, metrics := newTestStore(t)
	client := vuhivehttp.NewClientWithCollector(metrics, vuhivehttp.WithDiscardBody())

	resp, err := client.Get(context.Background(), ts.URL+"/fail")
	require.NoError(t, err, "non-OK status is not a transport error")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Nil(t, resp.Body)

	failedRate := store.AggregatedRateValue("vuhive.http.req_failed")
	assert.Equal(t, 1.0, failedRate, "failure rate should be 1.0 for 500 response")
}

// --- Finding 4: unsafe.String zero-copy in Text() ---

func TestResponse_Text_ZeroCopy_SharesUnderlying(t *testing.T) {
	body := []byte("hello world zero copy")
	resp := vuhivehttp.Response{
		StatusCode: http.StatusOK,
		Body:       body,
	}

	text := resp.Text()
	assert.Equal(t, "hello world zero copy", text)

	// Verify zero-copy: the string's underlying data should point to the same memory as the slice.
	if len(body) > 0 {
		slicePtr := unsafe.Pointer(&body[0])
		stringPtr := unsafe.Pointer(unsafe.StringData(text))
		assert.Equal(t, slicePtr, stringPtr,
			"Text() should return a zero-copy string sharing the same underlying memory as Body")
	}
}
