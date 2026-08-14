package metric

// Reader provides read-only inspection and summary aggregation across registered and recorded metrics.
type Reader interface {
	Registry
	Aggregator
}
