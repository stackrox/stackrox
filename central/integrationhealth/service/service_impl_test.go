package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingVulnDefsInfoProvider records the scanner type it was asked about so the
// test can assert which scanner GetVulnDefinitionsInfo resolves to.
type recordingVulnDefsInfoProvider struct {
	requestedType string
}

func (p *recordingVulnDefsInfoProvider) GetVulnDefsInfo(scannerType string) (*v1.VulnDefinitionsInfo, error) {
	p.requestedType = scannerType
	return &v1.VulnDefinitionsInfo{}, nil
}

func TestGetVulnDefinitionsInfo_ScannerTypeSelection(t *testing.T) {
	cases := map[string]struct {
		component            v1.VulnDefinitionsInfoRequest_Component
		legacyScannerEnabled bool
		expectedType         string
	}{
		"unspecified component defaults to Scanner V4": {
			component:    v1.VulnDefinitionsInfoRequest_SCANNER,
			expectedType: types.ScannerV4,
		},
		"explicit Scanner V4 resolves to Scanner V4": {
			component:    v1.VulnDefinitionsInfoRequest_SCANNER_V4,
			expectedType: types.ScannerV4,
		},
		"explicit legacy scanner resolves to V4 when legacy scanner is disabled": {
			component:    v1.VulnDefinitionsInfoRequest_SCANNER,
			expectedType: types.ScannerV4,
		},
		"explicit legacy scanner resolves to Clairify when legacy scanner is enabled": {
			component:            v1.VulnDefinitionsInfoRequest_SCANNER,
			legacyScannerEnabled: true,
			expectedType:         types.Clairify,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.LegacyScanner, tc.legacyScannerEnabled)

			provider := &recordingVulnDefsInfoProvider{}
			s := &serviceImpl{vulnDefsInfoProvider: provider}

			info, err := s.GetVulnDefinitionsInfo(context.Background(),
				&v1.VulnDefinitionsInfoRequest{Component: tc.component})
			require.NoError(t, err)
			assert.NotNil(t, info)
			assert.Equal(t, tc.expectedType, provider.requestedType)
		})
	}
}
