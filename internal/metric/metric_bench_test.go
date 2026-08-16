package metric_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/metric"
)

func BenchmarkCollector_Counter_Parallel(b *testing.B) {
	coll := metric.NewCollector(nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			coll.Counter("bench_counter", metric.Tags{}).Inc()
		}
	})
}

func BenchmarkCollector_Counter_PreResolved_Parallel(b *testing.B) {
	coll := metric.NewCollector(nil)
	c := coll.Counter("bench_counter_preresolved", metric.Tags{})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkCollector_Duration_Parallel(b *testing.B) {
	coll := metric.NewCollector(nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			coll.Duration("bench_duration", metric.Tags{}).Observe(5 * time.Millisecond)
		}
	})
}

func BenchmarkCollector_Gauge_Parallel(b *testing.B) {
	coll := metric.NewCollector(nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			coll.Gauge("bench_gauge", metric.Tags{}).Set(42.0)
		}
	})
}

func BenchmarkCollector_Rate_Parallel(b *testing.B) {
	coll := metric.NewCollector(nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			coll.Rate("bench_rate", metric.Tags{}).Add(1, 1)
		}
	})
}

func BenchmarkCollector_MakeKey_EmptyTags(b *testing.B) {
	tags := metric.Tags{}
	coll := metric.NewCollector(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.Counter("test_counter", tags).Inc()
	}
}

func BenchmarkCollector_MakeKey_SingleTag(b *testing.B) {
	tags := metric.Tags{"status": "200"}
	coll := metric.NewCollector(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.Counter("test_counter", tags).Inc()
	}
}

func BenchmarkCollector_MakeKey_MultipleTags(b *testing.B) {
	tags := metric.Tags{"status": "200", "method": "GET", "handler": "checkout"}
	coll := metric.NewCollector(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.Counter("test_counter", tags).Inc()
	}
}
