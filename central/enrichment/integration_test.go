package enrichment

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/cve/fetcher"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	integrationMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	nodeEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	vmEnricher "github.com/stackrox/rox/pkg/virtualmachine/enricher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestManagerUpsertAndRemove_VirtualMachineCategory(t *testing.T) {
	t.Run("upserts and removes explicit VM scanner integrations", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		imageIntegrationSet := integrationMocks.NewMockSet(ctrl)
		nodeEnricher := nodeEnricherMocks.NewMockNodeEnricher(ctrl)
		vmEnricher := &recordingVMEnricher{}
		cveFetcher := &recordingOrchestratorCVEManager{}

		manager := newManager(imageIntegrationSet, nodeEnricher, vmEnricher, cveFetcher)

		integration := &storage.ImageIntegration{
			Id:   "vm-scanner-id",
			Name: "Scanner V4",
			Type: scannerTypes.ScannerV4,
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_SCANNER,
				storage.ImageIntegrationCategory_VIRTUAL_MACHINE_SCANNER,
			},
			IntegrationConfig: &storage.ImageIntegration_ScannerV4{
				ScannerV4: &storage.ScannerV4Config{},
			},
		}

		imageIntegrationSet.EXPECT().UpdateImageIntegration(integration).Return(nil)
		nodeEnricher.EXPECT().RemoveNodeIntegration(integration.GetId())

		require.NoError(t, manager.Upsert(integration))
		require.Len(t, vmEnricher.upserts, 1)
		protoassert.Equal(t, integration, vmEnricher.upserts[0])
		assert.Equal(t, []string{integration.GetId()}, cveFetcher.removals)

		imageIntegrationSet.EXPECT().RemoveImageIntegration(integration.GetId()).Return(nil)
		nodeEnricher.EXPECT().RemoveNodeIntegration(integration.GetId())

		require.NoError(t, manager.Remove(integration.GetId()))
		assert.Equal(t, []string{integration.GetId()}, vmEnricher.removals)
	})

	t.Run("returns clear error for unsupported vm scanner category type", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		imageIntegrationSet := integrationMocks.NewMockSet(ctrl)
		nodeEnricher := nodeEnricherMocks.NewMockNodeEnricher(ctrl)
		vmEnricher := &recordingVMEnricher{
			upsertErr: fmt.Errorf("unsupported virtual machine scanner integration type: %q", scannerTypes.Quay),
		}

		manager := newManager(imageIntegrationSet, nodeEnricher, vmEnricher, &recordingOrchestratorCVEManager{})

		integration := &storage.ImageIntegration{
			Id:   "vm-scanner-id",
			Name: "Quay",
			Type: scannerTypes.Quay,
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_VIRTUAL_MACHINE_SCANNER,
			},
		}

		imageIntegrationSet.EXPECT().UpdateImageIntegration(integration).Return(nil)

		err := manager.Upsert(integration)
		require.Error(t, err)
		assert.Contains(t, err.Error(), scannerTypes.Quay)
	})
}

type recordingVMEnricher struct {
	upserts   []*storage.ImageIntegration
	removals  []string
	upsertErr error
}

func (*recordingVMEnricher) EnrichVirtualMachineWithVulnerabilities(*storage.VirtualMachine, *v4.IndexReport) error {
	return nil
}

func (r *recordingVMEnricher) UpsertVirtualMachineIntegration(integration *storage.ImageIntegration) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.upserts = append(r.upserts, integration)
	return nil
}

func (r *recordingVMEnricher) RemoveVirtualMachineIntegration(id string) {
	r.removals = append(r.removals, id)
}

var _ vmEnricher.VirtualMachineEnricher = (*recordingVMEnricher)(nil)

type recordingOrchestratorCVEManager struct {
	upserts  []*storage.OrchestratorIntegration
	removals []string
}

func (*recordingOrchestratorCVEManager) Start() {}

func (*recordingOrchestratorCVEManager) HandleClusterConnection() {}

func (*recordingOrchestratorCVEManager) GetAffectedClusters(context.Context, string, utils.CVEType, *cveMatcher.CVEMatcher) ([]*storage.Cluster, error) {
	return nil, nil
}

func (r *recordingOrchestratorCVEManager) UpsertOrchestratorIntegration(integration *storage.OrchestratorIntegration) error {
	r.upserts = append(r.upserts, integration)
	return nil
}

func (r *recordingOrchestratorCVEManager) RemoveIntegration(integrationID string) {
	r.removals = append(r.removals, integrationID)
}

var _ fetcher.OrchestratorIstioCVEManager = (*recordingOrchestratorCVEManager)(nil)
