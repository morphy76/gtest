package vuhive_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicVUContext_Group_SingleAndNested(t *testing.T) {
	yamlContent := `
version: "1.0"
default_scenario: group_journey
scenarios:
  group_journey:
    type: constant_vus
    vus: 1
    run_period: 100ms
    vu_timeout: 1s
    thresholds:
      - metric: "vuhive.group.01_Login.duration"
        stat: p95
        operator: "<"
        target: "500ms"
      - metric: "vuhive.group.02_Checkout::Payment.duration"
        stat: p95
        operator: "<"
        target: "500ms"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "vuhive.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	suite := vuhive.NewSuite("Group Journey Test")

	var capturedSummary vuhive.SummaryData
	var handleSummaryCalled bool

	suite.RegisterScenario("group_journey", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			// Top-level group
			if err := ctx.Group("01_Login", func(ctx vuhive.VUContext) error {
				time.Sleep(5 * time.Millisecond)
				return nil
			}); err != nil {
				return err
			}

			// Top-level group with nested group
			if err := ctx.Group("02_Checkout", func(ctx vuhive.VUContext) error {
				time.Sleep(5 * time.Millisecond)

				return ctx.Group("Payment", func(ctx vuhive.VUContext) error {
					time.Sleep(10 * time.Millisecond)
					return nil
				})
			}); err != nil {
				return err
			}

			return nil
		},
		HandleSummary: func(ctx vuhive.SummaryContext, summary vuhive.SummaryData) error {
			handleSummaryCalled = true
			capturedSummary = summary
			return nil
		},
	})

	var buf bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath}, &buf)
	require.NoError(t, res.Error)
	assert.True(t, res.Passed)
	assert.Equal(t, 0, res.ExitCode())

	out := buf.String()
	assert.Contains(t, out, "GROUPS")
	assert.Contains(t, out, "01_Login")
	assert.Contains(t, out, "02_Checkout")
	assert.Contains(t, out, "02_Checkout::Payment")

	require.True(t, handleSummaryCalled)
	require.Len(t, capturedSummary.Groups, 3)

	loginGrp := capturedSummary.Group("01_Login")
	require.NotNil(t, loginGrp)
	assert.Greater(t, loginGrp.Count, int64(0))

	checkoutGrp := capturedSummary.Group("02_Checkout")
	require.NotNil(t, checkoutGrp)
	assert.Greater(t, checkoutGrp.Count, int64(0))

	paymentGrp := capturedSummary.Group("02_Checkout::Payment")
	require.NotNil(t, paymentGrp)
	assert.Greater(t, paymentGrp.Count, int64(0))
}

func TestPublicVUContext_Group_ErrorPropagation(t *testing.T) {
	yamlContent := `
version: "1.0"
default_scenario: error_flow
scenarios:
  error_flow:
    type: constant_vus
    vus: 1
    run_period: 50ms
    vu_timeout: 1s
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "vuhive.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	suite := vuhive.NewSuite("Error Group Test")
	expectedErr := errors.New("group sub-action failed")

	suite.RegisterScenario("error_flow", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			err := ctx.Group("faulty_step", func(ctx vuhive.VUContext) error {
				return expectedErr
			})
			assert.ErrorIs(t, err, expectedErr)
			return fmt.Errorf("outer error: %w", err)
		},
	})

	var buf bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath}, &buf)
	require.NoError(t, res.Error)

	out := buf.String()
	assert.Contains(t, out, "faulty_step")
}
