// Package cli provides command-line flag parsing for gtest suite execution.
package cli

import (
	"flag"
	"fmt"
	"io"
)

// Flags holds the parsed command-line configuration options.
type Flags struct {
	ConfigPath    string
	ScenarioName  string
	LogLevel      string
	LogFormat     string
	ReportFormat  string
	ReportOut     string
	JSONReportOut string
	ShowVersion   bool
}

// ParseFlags parses command-line arguments into a Flags struct.
// It uses a local FlagSet to allow isolated unit testing without global state mutation.
func ParseFlags(args []string, errOutput io.Writer) (*Flags, error) {
	fs := flag.NewFlagSet("gtest", flag.ContinueOnError)
	if errOutput != nil {
		fs.SetOutput(errOutput)
	}

	flags := &Flags{}
	fs.StringVar(&flags.ConfigPath, "config", "gtest.yaml", "Path to the YAML configuration file")
	fs.StringVar(&flags.ScenarioName, "scenario", "", "Name of the scenario to execute")
	fs.StringVar(&flags.LogLevel, "log-level", "info", "Log verbosity: debug, info, warn, error")
	fs.StringVar(&flags.LogFormat, "log-format", "pretty", "Log output format: pretty or json")
	fs.StringVar(&flags.ReportFormat, "report-format", "console", "Report format: console or json")
	fs.StringVar(&flags.ReportOut, "report-out", "", "Write final report to this file path instead of stdout")
	fs.StringVar(&flags.JSONReportOut, "json-report-out", "", "Write JSON report document to this file path")
	fs.BoolVar(&flags.ShowVersion, "version", false, "Print library version and exit")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("gtest: invalid command line flags: %w", err)
	}

	return flags, nil
}
