package gtest

import "github.com/morphy76/gtest/internal/engine"

// ScenarioContext is the scoped execution context passed to every VU hook.
// It embeds context.Context so it can be passed directly to stdlib calls
// (http.NewRequestWithContext, etc.).
type ScenarioContext = engine.ScenarioContext
