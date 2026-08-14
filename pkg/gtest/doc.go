// Package gtest provides the public developer API and load test execution framework.
//
// gtest is an enterprise-grade Go load testing framework featuring:
//   - Declarative YAML configuration (VU count, pacing, ramp-up, ramp-down, SLA thresholds).
//   - Scenario lifecycle hooks (Setup, PreTest, RunVU, AfterTest, Teardown, HandleSummary).
//   - High-performance, lock-free telemetry (Counters, Gauges, Rates, and HDR Histograms).
//   - Non-blocking think time generators (fixed, uniform range, exponential, gaussian).
//   - Clean-architecture execution model with structured summary reports and JSON export.
//
// Basic Usage:
//
//	package main
//
//	import (
//		"net/http"
//		"time"
//
//		"github.com/morphy76/gtest/pkg/gtest"
//	)
//
//	func main() {
//		suite := gtest.NewSuite("E-Commerce Load Tests")
//
//		suite.RegisterScenario("checkout_flow", gtest.Scenario{
//			RunVU: func(ctx gtest.ScenarioContext) error {
//				start := time.Now()
//				resp, err := http.Get("https://api.example.com/checkout")
//				if err != nil {
//					return err
//				}
//				defer resp.Body.Close()
//
//				ctx.Metrics().Duration("checkout_latency", nil).Observe(time.Since(start))
//				ctx.Check("status_200", func() string {
//					if resp.StatusCode != http.StatusOK {
//						return "non-200 status code"
//					}
//					return ""
//				})
//				return nil
//			},
//		})
//
//		res := suite.Execute()
//		if res.ExitCode() != 0 {
//			// Handle failure or let main return non-zero exit code
//		}
//	}
package gtest
