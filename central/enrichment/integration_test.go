package enrichment

import (
	"errors"
	"fmt"
	"testing"

	cveFetcherMocks "github.com/stackrox/rox/central/cve/fetcher/mocks"
	"github.com/stackrox/rox/generated/storage"
	imageIntegrationMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	nodeEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protomock"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/uuid"
	virtualMachineEnricherMocks "github.com/stackrox/rox/pkg/virtualmachine/enricher/mocks"
	"github.com/stretchr/testify/assert"
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

func TestIsVirtualMachineIntegration(t *testing.T) {
	for name, tc := range map[string]struct {
		integration *storage.ImageIntegration
		expected    bool
	}{
		"nil integration": {
			integration: nil,
			expected:    false,
		},
		"nil categories": {
			integration: &storage.ImageIntegration{},
			expected:    false,
		},
		"empty categories": {
			integration: &storage.ImageIntegration{
				Categories: []storage.ImageIntegrationCategory{},
			},
			expected: false,
		},
		"other categories": {
			integration: &storage.ImageIntegration{
				Categories: []storage.ImageIntegrationCategory{
					storage.ImageIntegrationCategory_REGISTRY,
					storage.ImageIntegrationCategory_SCANNER,
					storage.ImageIntegrationCategory_NODE_SCANNER,
				},
			},
			expected: false,
		},
		"only VM category": {
			integration: &storage.ImageIntegration{
				Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_VIRTUAL_MACHINE_SCANNER},
			},
			expected: true,
		},
	} {
		t.Run(name, func(it *testing.T) {
			res := isVirtualMachineIntegration(tc.integration)
			assert.Equal(it, tc.expected, res)
		})
	}
}

func TestManagerUpsert(t *testing.T) {
	testErr := errors.New("test error")
	testIntegrationId1 := uuid.NewTestUUID(1).String()
	emptyIntegration := &storage.ImageIntegration{
		Id: testIntegrationId1,
	}
	goMockMatchEmptyIntegration := protomock.GoMockMatcherEqualMessage(emptyIntegration)
	testIntegrationId2 := uuid.NewTestUUID(2).String()
	virtualMachineIntegrationOnly := &storage.ImageIntegration{
		Id: testIntegrationId2,
		Categories: []storage.ImageIntegrationCategory{
			storage.ImageIntegrationCategory_VIRTUAL_MACHINE_SCANNER,
		},
	}
	goMockMatchVirtualMachineIntegrationOnly := protomock.GoMockMatcherEqualMessage(virtualMachineIntegrationOnly)
	for name, tc := range map[string]struct {
		integration   *storage.ImageIntegration
		setupMocks    func(*mockManager)
		expectedError error
	}{
		"image integration set update failure": {
			integration: nil,
			setupMocks: func(m *mockManager) {
				m.imageIntegrationSet.EXPECT().
					UpdateImageIntegration(nil).
					Return(testErr)
			},
			expectedError: testErr,
		},
		"image integration set update integration success": {
			integration: emptyIntegration,
			setupMocks: func(m *mockManager) {
				m.imageIntegrationSet.EXPECT().
					UpdateImageIntegration(goMockMatchEmptyIntegration).
					Return(nil)
				m.vmEnricher.EXPECT().
					RemoveVirtualMachineIntegration(emptyIntegration.GetId())
				m.nodeEnricher.EXPECT().
					RemoveNodeIntegration(emptyIntegration.GetId())
				m.cveFetcher.EXPECT().
					RemoveIntegration(emptyIntegration.GetId())
			},
			expectedError: nil,
		},
		"virtual machine only integration with successful upsert": {
			integration: virtualMachineIntegrationOnly,
			setupMocks: func(m *mockManager) {
				m.imageIntegrationSet.EXPECT().
					UpdateImageIntegration(goMockMatchVirtualMachineIntegrationOnly).
					Return(nil)
				m.vmEnricher.EXPECT().
					UpsertVirtualMachineIntegration(goMockMatchVirtualMachineIntegrationOnly).
					Return(nil)
				m.nodeEnricher.EXPECT().
					RemoveNodeIntegration(virtualMachineIntegrationOnly.GetId())
				m.cveFetcher.EXPECT().
					RemoveIntegration(virtualMachineIntegrationOnly.GetId())
			},
			expectedError: nil,
		},
		"virtual machine only integration with failing upsert": {
			integration: virtualMachineIntegrationOnly,
			setupMocks: func(m *mockManager) {
				m.imageIntegrationSet.EXPECT().
					UpdateImageIntegration(goMockMatchVirtualMachineIntegrationOnly).
					Return(nil)
				m.vmEnricher.EXPECT().
					UpsertVirtualMachineIntegration(goMockMatchVirtualMachineIntegrationOnly).
					Return(testErr)
			},
			expectedError: testErr,
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctrl := gomock.NewController(it)
			managerMockController := newMockManager(ctrl)
			if tc.setupMocks != nil {
				tc.setupMocks(managerMockController)
			}
			testManager := newManager(
				managerMockController.imageIntegrationSet,
				managerMockController.nodeEnricher,
				managerMockController.vmEnricher,
				managerMockController.cveFetcher,
			)
			err := testManager.Upsert(tc.integration)
			if tc.expectedError == nil {
				assert.NoError(it, err)
			} else {
				assert.ErrorIs(it, err, tc.expectedError)
			}
		})
	}
}

// region testHelpers
type mockManager struct {
	imageIntegrationSet *imageIntegrationMocks.MockSet
	nodeEnricher        *nodeEnricherMocks.MockNodeEnricher
	vmEnricher          *virtualMachineEnricherMocks.MockVirtualMachineEnricher
	cveFetcher          *cveFetcherMocks.MockOrchestratorIstioCVEManager
}

func newMockManager(ctrl *gomock.Controller) *mockManager {
	return &mockManager{
		imageIntegrationSet: imageIntegrationMocks.NewMockSet(ctrl),
		nodeEnricher:        nodeEnricherMocks.NewMockNodeEnricher(ctrl),
		vmEnricher:          virtualMachineEnricherMocks.NewMockVirtualMachineEnricher(ctrl),
		cveFetcher:          cveFetcherMocks.NewMockOrchestratorIstioCVEManager(ctrl),
	}
}

// endregion testHelpers
