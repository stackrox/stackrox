package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commonViews "github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/central/views/vmcve"
	cveViewMocks "github.com/stackrox/rox/central/views/vmcve/mocks"
	componentDSMocks "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/mocks"
	cveDSMocks "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore/mocks"
	scanDSMocks "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/mocks"
	vmDSMocks "github.com/stackrox/rox/central/virtualmachine/v2/datastore/mocks"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestGetVMDashboardCounts(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		setupMock     func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView)
		expectedError string
		expectedVM    int32
		expectedCVE   int32
	}{
		"successful counts": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().CountVirtualMachines(ctx, gomock.Any()).Return(5, nil)
				mockView.EXPECT().Count(ctx, gomock.Any()).Return(10, nil)
			},
			expectedVM:  5,
			expectedCVE: 10,
		},
		"vm count error": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().CountVirtualMachines(ctx, gomock.Any()).Return(0, errors.New("vm count failed"))
			},
			expectedError: "vm count failed",
		},
		"cve count error": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().CountVirtualMachines(ctx, gomock.Any()).Return(5, nil)
				mockView.EXPECT().Count(ctx, gomock.Any()).Return(0, errors.New("cve count failed"))
			},
			expectedError: "cve count failed",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVM := vmDSMocks.NewMockDataStore(ctrl)
			mockCVE := cveDSMocks.NewMockDataStore(ctrl)
			mockComp := componentDSMocks.NewMockDataStore(ctrl)
			mockScan := scanDSMocks.NewMockDataStore(ctrl)
			mockView := cveViewMocks.NewMockCveView(ctrl)

			service := &serviceImpl{
				vmDS:        mockVM,
				cveDS:       mockCVE,
				componentDS: mockComp,
				scanDS:      mockScan,
				cveView:     mockView,
			}

			tt.setupMock(mockVM, mockCVE, mockComp, mockScan, mockView)

			result, err := service.GetVMDashboardCounts(ctx, &v2.VMDashboardCountsRequest{})

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVM, result.GetVmCount())
				assert.Equal(t, tt.expectedCVE, result.GetCveCount())
			}
		})
	}
}

func TestGetVMVulnSummary(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		request       *v2.GetVMVulnSummaryRequest
		setupMock     func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView)
		expectedError string
	}{
		"empty id": {
			request: &v2.GetVMVulnSummaryRequest{
				Id: "",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
			},
			expectedError: "id must be specified",
		},
		"vm not found": {
			request: &v2.GetVMVulnSummaryRequest{
				Id: "vm-1",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().GetVirtualMachine(ctx, "vm-1").Return(nil, false, nil)
			},
			expectedError: "not found",
		},
		"successful with severity counts": {
			request: &v2.GetVMVulnSummaryRequest{
				Id: "vm-1",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().GetVirtualMachine(ctx, "vm-1").Return(&storage.VirtualMachineV2{
					Id:   "vm-1",
					Name: "test-vm",
				}, true, nil)
				mockView.EXPECT().CountBySeverity(ctx, gomock.Any()).Return(&commonViews.ResourceCountByImageCVESeverity{
					CriticalSeverityCount:         2,
					FixableCriticalSeverityCount:  1,
					ImportantSeverityCount:        3,
					FixableImportantSeverityCount: 2,
				}, nil)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVM := vmDSMocks.NewMockDataStore(ctrl)
			mockCVE := cveDSMocks.NewMockDataStore(ctrl)
			mockComp := componentDSMocks.NewMockDataStore(ctrl)
			mockScan := scanDSMocks.NewMockDataStore(ctrl)
			mockView := cveViewMocks.NewMockCveView(ctrl)

			service := &serviceImpl{
				vmDS:        mockVM,
				cveDS:       mockCVE,
				componentDS: mockComp,
				scanDS:      mockScan,
				cveView:     mockView,
			}

			tt.setupMock(mockVM, mockCVE, mockComp, mockScan, mockView)

			result, err := service.GetVMVulnSummary(ctx, tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.GetSeverityCounts())
			}
		})
	}
}

func TestListVMCVEsByVM(t *testing.T) {
	ctx := context.Background()

	cve1 := &storage.VirtualMachineCVEV2{
		Id:            "cve-uuid-1",
		VmV2Id:        "vm-1",
		VmComponentId: "comp-1",
		CveBaseInfo: &storage.CVEInfo{
			Cve:     "CVE-2024-1234",
			Summary: "test vuln 1",
			Link:    "https://example.com/1",
		},
		PreferredCvss:   7.5,
		Severity:        storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		IsFixable:       true,
		EpssProbability: 0.5,
	}

	tests := map[string]struct {
		request       *v2.ListVMCVEsByVMRequest
		setupMock     func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView)
		expectedError string
		expectedCount int32
	}{
		"empty vm_id": {
			request: &v2.ListVMCVEsByVMRequest{
				VmId: "",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
			},
			expectedError: "vm_id must be specified",
		},
		"successful list with CVEs": {
			request: &v2.ListVMCVEsByVMRequest{
				VmId: "vm-1",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockCVE.EXPECT().Count(ctx, gomock.Any()).Return(1, nil)
				mockCVE.EXPECT().SearchRawVMCVEs(ctx, gomock.Any()).Return([]*storage.VirtualMachineCVEV2{cve1}, nil)
			},
			expectedCount: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVM := vmDSMocks.NewMockDataStore(ctrl)
			mockCVE := cveDSMocks.NewMockDataStore(ctrl)
			mockComp := componentDSMocks.NewMockDataStore(ctrl)
			mockScan := scanDSMocks.NewMockDataStore(ctrl)
			mockView := cveViewMocks.NewMockCveView(ctrl)

			service := &serviceImpl{
				vmDS:        mockVM,
				cveDS:       mockCVE,
				componentDS: mockComp,
				scanDS:      mockScan,
				cveView:     mockView,
			}

			tt.setupMock(mockVM, mockCVE, mockComp, mockScan, mockView)

			result, err := service.ListVMCVEsByVM(ctx, tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, result.GetTotalCount())
			}
		})
	}
}

func TestGetVMCVEComponents(t *testing.T) {
	ctx := context.Background()

	cve1 := &storage.VirtualMachineCVEV2{
		Id:            "cve-uuid-1",
		VmV2Id:        "vm-1",
		VmComponentId: "comp-1",
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2024-1234",
		},
		HasFixedBy: &storage.VirtualMachineCVEV2_FixedBy{FixedBy: "1.2.3"},
		Advisory: &storage.Advisory{
			Name: "RHSA-2024:1234",
			Link: "https://access.redhat.com",
		},
	}

	comp1 := &storage.VirtualMachineComponentV2{
		Id:      "comp-1",
		Name:    "openssl",
		Version: "1.1.1",
		Source:  storage.SourceType_OS,
	}

	tests := map[string]struct {
		request       *v2.GetVMCVEComponentsRequest
		setupMock     func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView)
		expectedError string
	}{
		"empty vm_id": {
			request: &v2.GetVMCVEComponentsRequest{
				VmId:  "",
				CveId: "cve-1",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
			},
			expectedError: "vm_id and cve_id must be specified",
		},
		"empty cve_id": {
			request: &v2.GetVMCVEComponentsRequest{
				VmId:  "vm-1",
				CveId: "",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
			},
			expectedError: "vm_id and cve_id must be specified",
		},
		"successful with components": {
			request: &v2.GetVMCVEComponentsRequest{
				VmId:  "vm-1",
				CveId: "CVE-2024-1234",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockCVE.EXPECT().SearchRawVMCVEs(ctx, gomock.Any()).Return([]*storage.VirtualMachineCVEV2{cve1}, nil)
				mockComp.EXPECT().GetBatch(ctx, []string{"comp-1"}).Return([]*storage.VirtualMachineComponentV2{comp1}, nil)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVM := vmDSMocks.NewMockDataStore(ctrl)
			mockCVE := cveDSMocks.NewMockDataStore(ctrl)
			mockComp := componentDSMocks.NewMockDataStore(ctrl)
			mockScan := scanDSMocks.NewMockDataStore(ctrl)
			mockView := cveViewMocks.NewMockCveView(ctrl)

			service := &serviceImpl{
				vmDS:        mockVM,
				cveDS:       mockCVE,
				componentDS: mockComp,
				scanDS:      mockScan,
				cveView:     mockView,
			}

			tt.setupMock(mockVM, mockCVE, mockComp, mockScan, mockView)

			result, err := service.GetVMCVEComponents(ctx, tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestListVMComponents(t *testing.T) {
	ctx := context.Background()

	comp1 := &storage.VirtualMachineComponentV2{
		Id:       "comp-1",
		Name:     "openssl",
		Version:  "1.1.1",
		Source:   storage.SourceType_OS,
		CveCount: 2,
	}

	tests := map[string]struct {
		request       *v2.ListVMComponentsRequest
		setupMock     func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView)
		expectedError string
		expectedCount int32
	}{
		"empty vm_id": {
			request: &v2.ListVMComponentsRequest{
				VmId: "",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
			},
			expectedError: "vm_id must be specified",
		},
		"successful list": {
			request: &v2.ListVMComponentsRequest{
				VmId: "vm-1",
			},
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockComp.EXPECT().Count(ctx, gomock.Any()).Return(1, nil)
				mockComp.EXPECT().SearchRawVMComponents(ctx, gomock.Any()).Return([]*storage.VirtualMachineComponentV2{comp1}, nil)
			},
			expectedCount: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVM := vmDSMocks.NewMockDataStore(ctrl)
			mockCVE := cveDSMocks.NewMockDataStore(ctrl)
			mockComp := componentDSMocks.NewMockDataStore(ctrl)
			mockScan := scanDSMocks.NewMockDataStore(ctrl)
			mockView := cveViewMocks.NewMockCveView(ctrl)

			service := &serviceImpl{
				vmDS:        mockVM,
				cveDS:       mockCVE,
				componentDS: mockComp,
				scanDS:      mockScan,
				cveView:     mockView,
			}

			tt.setupMock(mockVM, mockCVE, mockComp, mockScan, mockView)

			result, err := service.ListVMComponents(ctx, tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, result.GetTotalCount())
			}
		})
	}
}

func TestListVMs(t *testing.T) {
	ctx := context.Background()

	vm1 := &storage.VirtualMachineV2{
		Id: "vm-1", Name: "test-vm-1", Namespace: "ns-1",
		ClusterId: "c-1", ClusterName: "cluster-1", GuestOs: "rhel-9",
		State: storage.VirtualMachineV2_RUNNING,
	}

	cve1 := &storage.VirtualMachineCVEV2{
		Id: "cve-uuid-1", VmV2Id: "vm-1",
		Severity:  storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		IsFixable: true,
	}

	comp1 := &storage.VirtualMachineComponentV2{
		Id: "comp-1", VmScanId: "scan-1", Name: "openssl", Version: "1.1.1",
	}
	compUnscanned := &storage.VirtualMachineComponentV2{
		Id: "comp-2", VmScanId: "scan-1", Name: "curl", Version: "7.0",
		Notes: []storage.VirtualMachineComponentV2_Note{storage.VirtualMachineComponentV2_UNSCANNED},
	}

	scan1 := &storage.VirtualMachineScanV2{
		Id: "scan-1", VmV2Id: "vm-1",
	}

	tests := map[string]struct {
		setupMock     func(*vmDSMocks.MockDataStore, *cveDSMocks.MockDataStore, *componentDSMocks.MockDataStore, *scanDSMocks.MockDataStore, *cveViewMocks.MockCveView)
		expectedError string
		expectedCount int32
		checkResult   func(t *testing.T, resp *v2.ListVMsResponse)
	}{
		"successful list with severity and component counts": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().CountVirtualMachines(ctx, gomock.Any()).Return(1, nil)
				mockVM.EXPECT().SearchRawVirtualMachines(ctx, gomock.Any()).Return([]*storage.VirtualMachineV2{vm1}, nil)
				mockCVE.EXPECT().SearchRawVMCVEs(ctx, gomock.Any()).Return([]*storage.VirtualMachineCVEV2{cve1}, nil)
				mockComp.EXPECT().SearchRawVMComponents(ctx, gomock.Any()).Return([]*storage.VirtualMachineComponentV2{comp1, compUnscanned}, nil)
				mockScan.EXPECT().GetBatch(ctx, gomock.Any()).Return([]*storage.VirtualMachineScanV2{scan1}, nil)
			},
			expectedCount: 1,
			checkResult: func(t *testing.T, resp *v2.ListVMsResponse) {
				require.Len(t, resp.GetVms(), 1)
				item := resp.GetVms()[0]
				assert.Equal(t, int32(1), item.GetCveSeverityCounts().GetCritical().GetTotal())
				assert.Equal(t, int32(1), item.GetCveSeverityCounts().GetCritical().GetFixable())
				assert.Equal(t, int32(2), item.GetComponentScanCount().GetTotal())
				assert.Equal(t, int32(1), item.GetComponentScanCount().GetScanned())
			},
		},
		"empty list": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().CountVirtualMachines(ctx, gomock.Any()).Return(0, nil)
				mockVM.EXPECT().SearchRawVirtualMachines(ctx, gomock.Any()).Return([]*storage.VirtualMachineV2{}, nil)
			},
			expectedCount: 0,
		},
		"vm count error": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockVM.EXPECT().CountVirtualMachines(ctx, gomock.Any()).Return(0, errors.New("count error"))
			},
			expectedError: "count error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVM := vmDSMocks.NewMockDataStore(ctrl)
			mockCVE := cveDSMocks.NewMockDataStore(ctrl)
			mockComp := componentDSMocks.NewMockDataStore(ctrl)
			mockScan := scanDSMocks.NewMockDataStore(ctrl)
			mockView := cveViewMocks.NewMockCveView(ctrl)

			svc := &serviceImpl{
				vmDS: mockVM, cveDS: mockCVE, componentDS: mockComp,
				scanDS: mockScan, cveView: mockView,
			}

			tt.setupMock(mockVM, mockCVE, mockComp, mockScan, mockView)

			result, err := svc.ListVMs(ctx, &v2.ListVMsRequest{})

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, result.GetTotalCount())
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

// mockCveCore implements vmcve.CveCore for testing ListVMCVEs.
type mockCveCore struct {
	cve        string
	cveIDs     []string
	topCVSS    float32
	affected   int
	epss       float32
	published  *time.Time
	discovered *time.Time
}

func (m *mockCveCore) GetCVE() string      { return m.cve }
func (m *mockCveCore) GetCVEIDs() []string { return m.cveIDs }
func (m *mockCveCore) GetVMsBySeverity() commonViews.ResourceCountByCVESeverity {
	return &commonViews.ResourceCountByImageCVESeverity{}
}
func (m *mockCveCore) GetTopCVSS() float32                    { return m.topCVSS }
func (m *mockCveCore) GetAffectedVMCount() int                { return m.affected }
func (m *mockCveCore) GetFirstDiscoveredInSystem() *time.Time { return m.discovered }
func (m *mockCveCore) GetPublishDate() *time.Time             { return m.published }
func (m *mockCveCore) GetEPSSProbability() float32            { return m.epss }

func TestListVMCVEs(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		setupMock     func(*vmDSMocks.MockDataStore, *cveDSMocks.MockDataStore, *componentDSMocks.MockDataStore, *scanDSMocks.MockDataStore, *cveViewMocks.MockCveView)
		expectedError string
		expectedCount int32
	}{
		"successful list": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockView.EXPECT().Count(ctx, gomock.Any()).Return(1, nil)
				mockView.EXPECT().Get(ctx, gomock.Any()).Return([]vmcve.CveCore{
					&mockCveCore{cve: "CVE-2024-1234", topCVSS: 7.5, affected: 3, epss: 0.5},
				}, nil)
				mockVM.EXPECT().CountVirtualMachines(ctx, gomock.Any()).Return(10, nil)
			},
			expectedCount: 1,
		},
		"view count error": {
			setupMock: func(mockVM *vmDSMocks.MockDataStore, mockCVE *cveDSMocks.MockDataStore, mockComp *componentDSMocks.MockDataStore, mockScan *scanDSMocks.MockDataStore, mockView *cveViewMocks.MockCveView) {
				mockView.EXPECT().Count(ctx, gomock.Any()).Return(0, errors.New("view error"))
			},
			expectedError: "view error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVM := vmDSMocks.NewMockDataStore(ctrl)
			mockCVE := cveDSMocks.NewMockDataStore(ctrl)
			mockComp := componentDSMocks.NewMockDataStore(ctrl)
			mockScan := scanDSMocks.NewMockDataStore(ctrl)
			mockView := cveViewMocks.NewMockCveView(ctrl)

			svc := &serviceImpl{
				vmDS: mockVM, cveDS: mockCVE, componentDS: mockComp,
				scanDS: mockScan, cveView: mockView,
			}

			tt.setupMock(mockVM, mockCVE, mockComp, mockScan, mockView)

			result, err := svc.ListVMCVEs(ctx, &v2.ListVMCVEsRequest{})

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, result.GetTotalCount())
			}
		})
	}
}
