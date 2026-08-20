//go:build kafka

package kafka

import (
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

func pubTags(topic string, status string) vuhive.Tags {
	return vuhive.Tags{
		"topic":  topic,
		"status": status,
	}
}

func subTags(topic string, group string, status string) vuhive.Tags {
	tags := vuhive.Tags{
		"topic":  topic,
		"status": status,
	}
	if group != "" {
		tags["group"] = group
	}
	return tags
}

func recordPubMetrics(collector vuhive.MetricsCollector, prefix string, topic string, duration time.Duration, bytesCount int, err error) {
	if collector == nil {
		return
	}

	status := "ok"
	if err != nil {
		status = "error"
	}
	tags := pubTags(topic, status)

	collector.Duration(prefix+MetricSuffixPubDuration, tags).Observe(duration)
	collector.Counter(prefix+MetricSuffixPubTotal, tags).Inc()

	if bytesCount > 0 {
		collector.Counter(prefix+MetricSuffixPubBytes, vuhive.Tags{"topic": topic}).Add(int64(bytesCount))
	}

	if err != nil {
		collector.Rate(prefix+MetricSuffixPubFailed, tags).Add(1, 1)
	} else {
		collector.Rate(prefix+MetricSuffixPubFailed, tags).Add(0, 1)
	}
}

func recordSubMetrics(collector vuhive.MetricsCollector, prefix string, topic string, group string, duration time.Duration, bytesCount int, err error) {
	if collector == nil {
		return
	}

	status := "ok"
	if err != nil {
		status = "error"
	}
	tags := subTags(topic, group, status)

	collector.Duration(prefix+MetricSuffixSubDuration, tags).Observe(duration)
	collector.Counter(prefix+MetricSuffixSubTotal, tags).Inc()

	if bytesCount > 0 {
		collector.Counter(prefix+MetricSuffixSubBytes, vuhive.Tags{"topic": topic}).Add(int64(bytesCount))
	}

	if err != nil {
		collector.Rate(prefix+MetricSuffixSubFailed, tags).Add(1, 1)
	} else {
		collector.Rate(prefix+MetricSuffixSubFailed, tags).Add(0, 1)
	}
}
