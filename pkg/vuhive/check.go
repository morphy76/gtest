package vuhive

// CheckFunc is a function that returns an empty string on pass,
// or a non-empty failure reason on failure.
type CheckFunc func() string
