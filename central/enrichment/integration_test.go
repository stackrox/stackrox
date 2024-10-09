package enrichment

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
)

func Test_ImageIntegrationToNodeIntegration(t *testing.T) {
	cases := map[string]struct {
		in               *storage.ImageIntegration
		expected         *storage.NodeIntegration
		expectedErrorMsg string
	}{
		"Valid v2": {
			in: &storage.ImageIntegration{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
				Type: scannerTypes.Clairify,
				Categories: []storage.ImageIntegrationCategory{
					storage.ImageIntegrationCategory_SCANNER,
					storage.ImageIntegrationCategory_NODE_SCANNER,
				},
				IntegrationConfig: &storage.ImageIntegration_Clairify{
					Clairify: &storage.ClairifyConfig{
						Endpoint: "https://localhost:8080",
					},
				},
			},
			expected: &storage.NodeIntegration{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
				Type: scannerTypes.Clairify,
				IntegrationConfig: &storage.NodeIntegration_Clairify{
					Clairify: &storage.ClairifyConfig{
						Endpoint: "https://localhost:8080",
					},
				},
			},
			expectedErrorMsg: "",
		},
		"Valid v4": {
			in: &storage.ImageIntegration{
				Id:   "a87471e6-9678-4e66-8348-91e302b6de07",
				Name: "Scanner V4",
				Type: scannerTypes.ScannerV4,
				Categories: []storage.ImageIntegrationCategory{
					storage.ImageIntegrationCategory_SCANNER,
					storage.ImageIntegrationCategory_NODE_SCANNER,
				},
				IntegrationConfig: &storage.ImageIntegration_ScannerV4{
					ScannerV4: &storage.ScannerV4Config{
						IndexerEndpoint: "https://localhost:8443",
						MatcherEndpoint: "https://localhost:9443",
					},
				},
			},
			expected: &storage.NodeIntegration{
				Id:   "a87471e6-9678-4e66-8348-91e302b6de07",
				Name: "Scanner V4",
				Type: scannerTypes.ScannerV4,
				IntegrationConfig: &storage.NodeIntegration_Scannerv4{
					Scannerv4: &storage.ScannerV4Config{
						IndexerEndpoint: "https://localhost:8443",
						MatcherEndpoint: "https://localhost:9443",
					},
				},
			},
			expectedErrorMsg: "",
		},
		"Invalid Scanner Type": {
			in: &storage.ImageIntegration{
				Id:   "a87471e6-0000-0000-0000-91e302b6de07",
				Name: "Quay",
				Type: scannerTypes.Quay,
			},
			expectedErrorMsg: fmt.Sprintf("unsupported integration type: %q.", scannerTypes.Quay),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := ImageIntegrationToNodeIntegration(c.in)

			if c.expectedErrorMsg != "" {
				assert.ErrorContains(t, err, c.expectedErrorMsg)
			} else {
				protoassert.Equal(t, c.expected, actual)
				assert.NoError(t, err)
			}
		})
	}
}
