package http

import (
	"net/http"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

// Client is an instrumented HTTP client that automatically records latency,
// status code counters, and error rates for every request executed.
// Client is safe for concurrent use from multiple VU goroutines.
type Client struct {
	inner   *http.Client
	cfg     clientConfig
	metrics vuhive.MetricsCollector
}

// NewClient creates an instrumented HTTP client from a SetupContext.
// The MetricsCollector is extracted from the context for automatic metric recording.
// Options configure timeouts, headers, TLS settings, and transport parameters.
func NewClient(ctx vuhive.SetupContext, opts ...Option) *Client {
	return newClientWithMetrics(ctx.Metrics(), opts...)
}

// NewClientFromVU creates an instrumented HTTP client from a VUContext.
// Use this when you need a per-VU client instance rather than a shared client created in Setup.
func NewClientFromVU(ctx vuhive.VUContext, opts ...Option) *Client {
	return newClientWithMetrics(ctx.Metrics(), opts...)
}

// NewClientWithCollector creates an instrumented HTTP client from a MetricsCollector directly.
// This is primarily useful for testing or for constructing a client outside of a scenario context.
func NewClientWithCollector(metrics vuhive.MetricsCollector, opts ...Option) *Client {
	return newClientWithMetrics(metrics, opts...)
}

func newClientWithMetrics(metrics vuhive.MetricsCollector, opts ...Option) *Client {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Client{
		inner: &http.Client{
			Timeout:   cfg.timeout,
			Transport: cfg.buildTransport(),
		},
		cfg:     cfg,
		metrics: metrics,
	}
}
