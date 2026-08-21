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

// BaseURL returns the configured base URL for this client.
func (c *Client) BaseURL() string {
	return c.cfg.baseURL
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

// NewClientFromConfig creates an instrumented HTTP client initialized from the SetupContext's
// declarative HTTP configuration (vuhive.yaml), with optional programmatic Option overrides.
func NewClientFromConfig(ctx vuhive.SetupContext, opts ...Option) *Client {
	return newClientFromHTTPConfig(ctx.HTTPConfig(), ctx.Metrics(), opts...)
}

// NewClientFromVUConfig creates an instrumented HTTP client initialized from the VUContext's
// declarative HTTP configuration (vuhive.yaml), with optional programmatic Option overrides.
func NewClientFromVUConfig(ctx vuhive.VUContext, opts ...Option) *Client {
	return newClientFromHTTPConfig(ctx.HTTPConfig(), ctx.Metrics(), opts...)
}

func newClientFromHTTPConfig(httpCfg vuhive.HTTPConfig, metrics vuhive.MetricsCollector, opts ...Option) *Client {
	cfg := defaultConfig()
	if httpCfg.BaseURL != "" {
		cfg.baseURL = httpCfg.BaseURL
	}
	if httpCfg.Timeout > 0 {
		cfg.timeout = httpCfg.Timeout
	}
	if len(httpCfg.Headers) > 0 {
		if cfg.defaultHeaders == nil {
			cfg.defaultHeaders = make(map[string]string, len(httpCfg.Headers))
		}
		for k, v := range httpCfg.Headers {
			cfg.defaultHeaders[k] = v
		}
	}
	if httpCfg.TLS.InsecureSkipVerify {
		cfg.tlsInsecureSkipVerify = true
	}
	if httpCfg.Pool.MaxIdleConns > 0 {
		cfg.maxIdleConns = httpCfg.Pool.MaxIdleConns
	}
	if httpCfg.Pool.MaxIdleConnsPerHost > 0 {
		cfg.maxIdleConnsPerHost = httpCfg.Pool.MaxIdleConnsPerHost
	}
	if httpCfg.Pool.IdleConnTimeout > 0 {
		cfg.idleConnTimeout = httpCfg.Pool.IdleConnTimeout
	}
	if httpCfg.DetailedTiming {
		cfg.detailedTiming = true
	}
	if httpCfg.MetricPrefix != "" {
		cfg.metricPrefix = httpCfg.MetricPrefix
	}

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

