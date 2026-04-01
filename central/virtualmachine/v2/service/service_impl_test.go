package service

import (
	"context"
	"errors"
	"testing"

	cveViewMocks "github.com/stackrox/rox/central/views/vmcve/mocks"
	componentDSMocks "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/mocks"
	cveDSMocks "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore/mocks"
	scanDSMocks "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/mocks"
	vmDSMocks "github.com/stackrox/rox/central/virtualmachine/v2/datastore/mocks"
	v2 "github.com/stackrox/rox/generated/api/v2"
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
