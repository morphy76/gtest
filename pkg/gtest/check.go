package gtest

import "github.com/morphy76/gtest/internal/engine"

// CheckFunc is a function that returns an empty string on pass,
// or a non-empty failure reason on failure.
type CheckFunc = engine.CheckFunc
