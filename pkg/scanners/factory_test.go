package scanners

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactory_ClairifyRegistration(t *testing.T) {
	clairifyIntegration := &storage.ImageIntegration{
		Id:   "test-clairify",
		Name: "Test Clairify",
		Type: scannerTypes.Clairify,
		IntegrationConfig: &storage.ImageIntegration_Clairify{
			Clairify: &storage.ClairifyConfig{
				Endpoint: "https://localhost:8080",
			},
		},
	}

	tests := map[string]struct {
		legacyScannerEnabled bool
		expectDoesNotExist   bool
	}{
		"when LegacyScanner is enabled, Clairify creator is registered": {
			legacyScannerEnabled: true,
			expectDoesNotExist:   false,
		},
		"when LegacyScanner is disabled, Clairify creator is not registered": {
			legacyScannerEnabled: false,
			expectDoesNotExist:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.LegacyScanner, tc.legacyScannerEnabled)

			factory := NewFactory(nil)
			_, err := factory.CreateScanner(clairifyIntegration)

			if tc.expectDoesNotExist {
				assert.ErrorContains(t, err, fmt.Sprintf("scanner with type %q does not exist", scannerTypes.Clairify))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
