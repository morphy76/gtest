package runner

import (
	"errors"
	"fmt"
	"io"

	"github.com/morphy76/vuhive/internal/cli"
	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/engine"
	"github.com/morphy76/vuhive/internal/version"
)

// ScenarioNotFoundError indicates a scenario was not found in the config or not registered.
type ScenarioNotFoundError struct {
	Name    string
	Message string
}

func (e *ScenarioNotFoundError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("vuhive: scenario %q not found: %s", e.Name, e.Message)
	}
	return fmt.Sprintf("vuhive: scenario not found: %s", e.Message)
}

var _ error = (*ScenarioNotFoundError)(nil)

// ScenarioRegistry provides access to named scenarios.
type ScenarioRegistry interface {
	Name() string
	GetScenario(name string) (engine.Scenario, bool)
}

// ResolvedScenario represents the fully validated scenario and configuration ready for execution.
type ResolvedScenario struct {
	Flags          *cli.Flags
	Config         *config.Config
	TargetScenario string
	Scenario       engine.Scenario
	ScenarioCfg    config.ScenarioConfig
	ShowVersion    bool
}

// ScenarioResolver encapsulates flag parsing, config loading, and scenario validation logic.
type ScenarioResolver struct {
	registry ScenarioRegistry
}

// NewScenarioResolver creates a new ScenarioResolver for a given ScenarioRegistry.
func NewScenarioResolver(registry ScenarioRegistry) *ScenarioResolver {
	return &ScenarioResolver{
		registry: registry,
	}
}

// Resolve parses CLI flags, loads config, and validates the scenario against the registry and config file.
func (r *ScenarioResolver) Resolve(args []string, stdout io.Writer) (*ResolvedScenario, error) {
	flags, err := cli.ParseFlags(args, stdout)
	if err != nil {
		return nil, &config.ConfigError{Err: err}
	}

	if flags.ShowVersion {
		if _, err := fmt.Fprintf(stdout, "vuhive version %s (commit: %s, build_time: %s)\n",
			version.Version, version.Commit, version.BuildTime); err != nil {
			return nil, err
		}
		return &ResolvedScenario{
			Flags:       flags,
			ShowVersion: true,
		}, nil
	}

	cfg, err := config.LoadFromFile(flags.ConfigPath)
	if err != nil {
		var valErr *config.ValidationError
		if errors.As(err, &valErr) {
			return nil, valErr
		}
		var cfgErr *config.ConfigError
		if errors.As(err, &cfgErr) {
			return nil, cfgErr
		}
		return nil, &config.ConfigError{Path: flags.ConfigPath, Err: err}
	}

	targetScenario := flags.ScenarioName
	if targetScenario == "" {
		targetScenario = cfg.DefaultScenario
	}
	if targetScenario == "" {
		return nil, &ScenarioNotFoundError{
			Name:    "",
			Message: "no scenario specified via --scenario flag or default_scenario in config",
		}
	}

	scenarioCfg, inConfig := cfg.Scenarios[targetScenario]
	scenario, registered := r.registry.GetScenario(targetScenario)

	if !registered {
		return nil, &ScenarioNotFoundError{
			Name:    targetScenario,
			Message: fmt.Sprintf("scenario %q is not registered in Suite", targetScenario),
		}
	}
	if !inConfig {
		return nil, &ScenarioNotFoundError{
			Name:    targetScenario,
			Message: fmt.Sprintf("scenario %q is registered in code but not defined in config file %q", targetScenario, flags.ConfigPath),
		}
	}

	return &ResolvedScenario{
		Flags:          flags,
		Config:         cfg,
		TargetScenario: targetScenario,
		Scenario:       scenario,
		ScenarioCfg:    scenarioCfg,
	}, nil
}

// resolveScenario is a package-level helper that invokes ScenarioResolver.Resolve.
func resolveScenario(registry ScenarioRegistry, args []string, stdout io.Writer) (*ResolvedScenario, error) {
	return NewScenarioResolver(registry).Resolve(args, stdout)
}
