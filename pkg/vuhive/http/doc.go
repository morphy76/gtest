// Package http provides an instrumented HTTP client for the vuhive load testing framework.
//
// The Client automatically records latency, status code counters, and error rates for
// every request, eliminating boilerplate metric recording from RunVU hooks.
//
// Basic usage:
//
//	// In Setup: create a shared instrumented client
//	client := vuhivehttp.NewClient(ctx,
//	    vuhivehttp.WithTimeout(5*time.Second),
//	    vuhivehttp.WithHeader("Authorization", "Bearer "+token),
//	)
//
//	// In RunVU: execute requests — metrics are recorded automatically
//	resp, err := client.Get(ctx, serverURL+"/api/checkout")
//	if err != nil {
//	    return err
//	}
//	var result CheckoutResult
//	if err := resp.JSON(&result); err != nil {
//	    return err
//	}
//
// Automatic metrics recorded per request:
//   - vuhive.http.req_duration (Duration): total request latency
//   - vuhive.http.req_failed (Rate): failed vs. total request ratio
//   - vuhive.http.reqs (Counter): total request count
//
// Opt-in phase-breakdown metrics (enabled via WithDetailedTiming):
//   - vuhive.http.req_connecting (Duration): TCP connection establishment time
//   - vuhive.http.req_tls_handshaking (Duration): TLS handshake time
//   - vuhive.http.req_sending (Duration): request write time
//   - vuhive.http.req_receiving (Duration): response read time
package http
