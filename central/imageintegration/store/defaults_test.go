package store

import (
	"testing"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDelayedIntegrations(t *testing.T) {
	tests := map[string]struct {
		legacyScannerEnabled bool
		expectCount          int
	}{
		"when LegacyScanner is enabled, returns Clairify scanner": {
			legacyScannerEnabled: true,
			expectCount:          1,
		},
		"when LegacyScanner is disabled, returns empty list": {
			legacyScannerEnabled: false,
			expectCount:          0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.LegacyScanner, tc.legacyScannerEnabled)

			integrations := GetDelayedIntegrations()

			if tc.expectCount == 0 {
				assert.Empty(t, integrations)
			} else {
				require.Len(t, integrations, tc.expectCount)
				assert.Equal(t, defaultScanner.GetName(), integrations[0].Integration.GetName())
			}
		})
	}
}
