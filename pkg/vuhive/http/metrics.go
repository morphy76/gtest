package http

import (
	"crypto/tls"
	"net/http/httptrace"
	"strconv"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

const defaultMetricPrefix = "vuhive.http."

// Built-in HTTP metric name suffixes, appended to the configured metric prefix.
const (
	// MetricSuffixReqDuration is the suffix for the total request latency histogram.
	MetricSuffixReqDuration = "req_duration"

	// MetricSuffixReqFailed is the suffix for the request failure rate.
	MetricSuffixReqFailed = "req_failed"

	// MetricSuffixReqs is the suffix for the total request counter.
	MetricSuffixReqs = "reqs"

	// MetricSuffixReqConnecting is the suffix for TCP connection time (opt-in).
	MetricSuffixReqConnecting = "req_connecting"

	// MetricSuffixReqTLSHandshaking is the suffix for TLS handshake time (opt-in).
	MetricSuffixReqTLSHandshaking = "req_tls_handshaking"

	// MetricSuffixReqSending is the suffix for request write time (opt-in).
	MetricSuffixReqSending = "req_sending"

	// MetricSuffixReqReceiving is the suffix for response read time (opt-in).
	MetricSuffixReqReceiving = "req_receiving"
)

// requestTags builds metric tags for a request.
func requestTags(method, url string, statusCode int) vuhive.Tags {
	return vuhive.Tags{
		"method": method,
		"url":    url,
		"status": strconv.Itoa(statusCode),
	}
}

// requestTagsNoStatus builds metric tags for a request that failed before receiving a response.
func requestTagsNoStatus(method, url string) vuhive.Tags {
	return vuhive.Tags{
		"method": method,
		"url":    url,
		"status": "0",
	}
}

// traceTimings captures HTTP phase timings from httptrace callbacks.
type traceTimings struct {
	connectStart    time.Time
	connectDone     time.Time
	tlsStart        time.Time
	tlsDone         time.Time
	wroteHeaders    time.Time
	gotFirstByte    time.Time

	connectDuration time.Duration
	tlsDuration     time.Duration
}

// newClientTrace creates an httptrace.ClientTrace that populates the given traceTimings.
func newClientTrace(t *traceTimings) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) {
			t.connectStart = time.Now()
		},
		ConnectDone: func(_, _ string, err error) {
			if err == nil {
				t.connectDone = time.Now()
				t.connectDuration = t.connectDone.Sub(t.connectStart)
			}
		},
		TLSHandshakeStart: func() {
			t.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			t.tlsDone = time.Now()
			t.tlsDuration = t.tlsDone.Sub(t.tlsStart)
		},
		WroteHeaders: func() {
			t.wroteHeaders = time.Now()
		},
		GotFirstResponseByte: func() {
			t.gotFirstByte = time.Now()
		},
	}
}

// recordMetrics records all standard request metrics.
func (c *Client) recordMetrics(method, url string, statusCode int, totalDuration time.Duration, failed bool) {
	tags := requestTags(method, url, statusCode)

	c.metrics.Duration(c.cfg.metricPrefix+MetricSuffixReqDuration, tags).Observe(totalDuration)
	c.metrics.Counter(c.cfg.metricPrefix+MetricSuffixReqs, tags).Inc()

	if failed {
		c.metrics.Rate(c.cfg.metricPrefix+MetricSuffixReqFailed, tags).Add(1, 1)
	} else {
		c.metrics.Rate(c.cfg.metricPrefix+MetricSuffixReqFailed, tags).Add(0, 1)
	}
}

// recordFailedMetrics records metrics when a request fails before receiving a response.
func (c *Client) recordFailedMetrics(method, url string, totalDuration time.Duration) {
	tags := requestTagsNoStatus(method, url)

	c.metrics.Duration(c.cfg.metricPrefix+MetricSuffixReqDuration, tags).Observe(totalDuration)
	c.metrics.Counter(c.cfg.metricPrefix+MetricSuffixReqs, tags).Inc()
	c.metrics.Rate(c.cfg.metricPrefix+MetricSuffixReqFailed, tags).Add(1, 1)
}

// recordDetailedTimings records opt-in phase timing metrics.
func (c *Client) recordDetailedTimings(tags vuhive.Tags, timings *traceTimings, sendingDuration, receivingDuration time.Duration) {
	if timings.connectDuration > 0 {
		c.metrics.Duration(c.cfg.metricPrefix+MetricSuffixReqConnecting, tags).Observe(timings.connectDuration)
	}
	if timings.tlsDuration > 0 {
		c.metrics.Duration(c.cfg.metricPrefix+MetricSuffixReqTLSHandshaking, tags).Observe(timings.tlsDuration)
	}
	if sendingDuration > 0 {
		c.metrics.Duration(c.cfg.metricPrefix+MetricSuffixReqSending, tags).Observe(sendingDuration)
	}
	if receivingDuration > 0 {
		c.metrics.Duration(c.cfg.metricPrefix+MetricSuffixReqReceiving, tags).Observe(receivingDuration)
	}
}
