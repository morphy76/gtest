package metric_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator_GroupSummaries_Empty(t *testing.T) {
	store := metric.NewStore()
	summaries := store.GroupSummaries()
	assert.Nil(t, summaries)
}

func TestAggregator_GroupSummaries_SingleAndNested(t *testing.T) {
	store := metric.NewStore()

	loginMetric := metric.GroupMetricName("01_Login")
	assert.Equal(t, "vuhive.group.01_Login.duration", loginMetric)

	checkoutMetric := metric.GroupMetricName("03_Checkout")
	assert.Equal(t, "vuhive.group.03_Checkout.duration", checkoutMetric)

	nestedPaymentMetric := metric.GroupMetricName("03_Checkout::Submit_Payment")
	assert.Equal(t, "vuhive.group.03_Checkout::Submit_Payment.duration", nestedPaymentMetric)

	loginHist := store.Duration(loginMetric, nil)
	loginHist.Observe(10 * time.Millisecond)
	loginHist.Observe(20 * time.Millisecond)
	loginHist.Observe(30 * time.Millisecond)

	checkoutHist := store.Duration(checkoutMetric, nil)
	checkoutHist.Observe(100 * time.Millisecond)

	paymentHist := store.Duration(nestedPaymentMetric, nil)
	paymentHist.Observe(50 * time.Millisecond)
	paymentHist.Observe(60 * time.Millisecond)

	// Also record non-group metrics to verify they are ignored by GroupSummaries
	store.Counter("vuhive.checks.passed", nil).Inc()
	store.Duration("custom_request_duration", nil).Observe(5 * time.Millisecond)

	summaries := store.GroupSummaries()
	require.Len(t, summaries, 3)

	// Summaries should be sorted alphabetically by group path
	assert.Equal(t, "01_Login", summaries[0].Name)
	assert.Equal(t, int64(3), summaries[0].Count)
	assertDurationWithinTolerance(t, 10*time.Millisecond, summaries[0].Min, 500*time.Microsecond, "min")
	assertDurationWithinTolerance(t, 30*time.Millisecond, summaries[0].Max, 500*time.Microsecond, "max")

	assert.Equal(t, "03_Checkout", summaries[1].Name)
	assert.Equal(t, int64(1), summaries[1].Count)
	assertDurationWithinTolerance(t, 100*time.Millisecond, summaries[1].Min, 1*time.Millisecond, "min")
	assertDurationWithinTolerance(t, 100*time.Millisecond, summaries[1].Max, 1*time.Millisecond, "max")

	assert.Equal(t, "03_Checkout::Submit_Payment", summaries[2].Name)
	assert.Equal(t, int64(2), summaries[2].Count)
	assertDurationWithinTolerance(t, 50*time.Millisecond, summaries[2].Min, 1*time.Millisecond, "min")
	assertDurationWithinTolerance(t, 60*time.Millisecond, summaries[2].Max, 1*time.Millisecond, "max")

}
