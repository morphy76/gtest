package gtest

import "github.com/morphy76/gtest/internal/engine"

// ExecutionIdentity provides execution identity attributes (VU ID, iteration index, and scenario name).
type ExecutionIdentity = engine.ExecutionIdentity

// ConfigProvider provides access to scenario configuration parameters.
type ConfigProvider = engine.ConfigProvider

// StateProvider provides read-only access to global scenario state returned by Setup.
type StateProvider = engine.StateProvider

// ObservabilityProvider provides access to structured logging and metric collection.
type ObservabilityProvider = engine.ObservabilityProvider

// WorkflowController provides workflow execution controls such as delays and inline assertions.
type WorkflowController = engine.WorkflowController

// ScenarioContext is the scoped execution context passed to every VU hook.
// It embeds context.Context and composes focused capability interfaces.
type ScenarioContext = engine.ScenarioContext

