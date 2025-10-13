package telemetry

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGather(t *testing.T) {
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	t.Setenv(features.ComplianceEnhancements.EnvVar(), "false")

	props, err := Gather(context.Background())
	require.NoError(t, err)

	expectedProps := map[string]any{
		"Feature ROX_SCANNER_V4":              true,
		"Feature ROX_COMPLIANCE_ENHANCEMENTS": false,
	}
	assert.Subset(t, props, expectedProps)
}
