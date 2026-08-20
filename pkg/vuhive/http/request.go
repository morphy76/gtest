package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"
)

// Get performs an HTTP GET request and returns an instrumented Response.
// Metrics are automatically recorded including latency, request count, and failure rate.
func (c *Client) Get(ctx context.Context, rawURL string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("vuhive/http: failed to create GET request: %w", err)
	}
	return c.Do(ctx, req)
}

// Post performs an HTTP POST request with the given content type and body.
// Metrics are automatically recorded including latency, request count, and failure rate.
func (c *Client) Post(ctx context.Context, rawURL string, contentType string, body io.Reader) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("vuhive/http: failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// Put performs an HTTP PUT request with the given content type and body.
// Metrics are automatically recorded including latency, request count, and failure rate.
func (c *Client) Put(ctx context.Context, rawURL string, contentType string, body io.Reader) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("vuhive/http: failed to create PUT request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// Delete performs an HTTP DELETE request.
// Metrics are automatically recorded including latency, request count, and failure rate.
func (c *Client) Delete(ctx context.Context, rawURL string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("vuhive/http: failed to create DELETE request: %w", err)
	}
	return c.Do(ctx, req)
}

// Do executes an arbitrary HTTP request and returns an instrumented Response.
// Default headers configured via WithHeader/WithHeaders are added to the request.
// Metrics are automatically recorded for every call.
func (c *Client) Do(ctx context.Context, req *http.Request) (*Response, error) {
	// Apply default headers (do not overwrite headers already set on the request).
	for k, v := range c.cfg.defaultHeaders {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	method := req.Method
	metricURL := sanitizeURL(req.URL)

	var timings traceTimings
	if c.cfg.detailedTiming {
		traceCtx := httptrace.WithClientTrace(ctx, newClientTrace(&timings))
		req = req.WithContext(traceCtx)
	} else if req.Context() != ctx {
		req = req.WithContext(ctx)
	}

	start := time.Now()
	resp, err := c.inner.Do(req)
	totalDuration := time.Since(start)

	if err != nil {
		c.recordFailedMetrics(method, metricURL, totalDuration)
		return nil, fmt.Errorf("vuhive/http: request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	readDone := time.Now()

	if readErr != nil {
		c.recordFailedMetrics(method, metricURL, time.Since(start))
		return nil, fmt.Errorf("vuhive/http: failed to read response body: %w", readErr)
	}

	failed := resp.StatusCode < 200 || resp.StatusCode >= 400
	c.recordMetrics(method, metricURL, resp.StatusCode, totalDuration, failed)

	if c.cfg.detailedTiming {
		tags := requestTags(method, metricURL, resp.StatusCode)

		var sendingDuration time.Duration
		if !timings.wroteHeaders.IsZero() && !timings.gotFirstByte.IsZero() {
			sendingDuration = timings.gotFirstByte.Sub(timings.wroteHeaders)
		}

		var receivingDuration time.Duration
		if !timings.gotFirstByte.IsZero() {
			receivingDuration = readDone.Sub(timings.gotFirstByte)
		}

		c.recordDetailedTimings(tags, &timings, sendingDuration, receivingDuration)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       bodyBytes,
	}, nil
}

// sanitizeURL extracts the path from a URL for metric tagging,
// stripping query parameters and fragments to prevent high-cardinality tags.
func sanitizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Path
}
