package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/convert/storagetov2"
	datastoreMocks "github.com/stackrox/rox/central/virtualmachine/datastore/mocks"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestGetVirtualMachine(t *testing.T) {
	ctx := context.Background()

	storedTestVM := &storage.VirtualMachine{
		Id:        "test-vm-id",
		Name:      "test-vm",
		Namespace: "test-namespace",
	}

	testVM := storagetov2.VirtualMachine(storedTestVM)

	tests := []struct {
		name           string
		request        *v2.GetVirtualMachineRequest
		setupMock      func(*datastoreMocks.MockDataStore)
		expectedResult *v2.VirtualMachine
		expectedError  string
	}{
		{
			name: "successful get",
			request: &v2.GetVirtualMachineRequest{
				Id: "test-vm-id",
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					GetVirtualMachine(ctx, "test-vm-id").
					Return(storedTestVM, true, nil)
			},
			expectedResult: testVM,
			expectedError:  "",
		},
		{
			name: "empty ID",
			request: &v2.GetVirtualMachineRequest{
				Id: "",
			},
			setupMock:      func(mockDS *datastoreMocks.MockDataStore) {},
			expectedResult: nil,
			expectedError:  "id must be specified",
		},
		{
			name: "virtual machine not found",
			request: &v2.GetVirtualMachineRequest{
				Id: "non-existent-vm",
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					GetVirtualMachine(ctx, "non-existent-vm").
					Return(nil, false, nil)
			},
			expectedResult: nil,
			expectedError:  "does not exist",
		},
		{
			name: "datastore error",
			request: &v2.GetVirtualMachineRequest{
				Id: "test-vm-id",
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					GetVirtualMachine(ctx, "test-vm-id").
					Return(nil, false, errors.New("datastore error"))
			},
			expectedResult: nil,
			expectedError:  "datastore error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDatastore := datastoreMocks.NewMockDataStore(ctrl)

			service := &serviceImpl{
				datastore: mockDatastore,
			}

			tt.setupMock(mockDatastore)

			result, err := service.GetVirtualMachine(ctx, tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				protoassert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestListVirtualMachines(t *testing.T) {
	ctx := context.Background()

	storageVMs := []*storage.VirtualMachine{
		{
			Id:        "vm-1",
			Name:      "test-vm-1",
			Namespace: "namespace-1",
		},
		{
			Id:        "vm-2",
			Name:      "test-vm-2",
			Namespace: "namespace-2",
		},
	}

	testVMs := make([]*v2.VirtualMachine, len(storageVMs))
	for i, vm := range storageVMs {
		testVMs[i] = storagetov2.VirtualMachine(vm)
	}

	storageVMsInReversedOrder := make([]*storage.VirtualMachine, len(storageVMs))
	for i, vm := range storageVMs {
		storageVMsInReversedOrder[len(storageVMsInReversedOrder)-i-1] = vm
	}

	testVMsInReversedOrder := make([]*v2.VirtualMachine, len(testVMs))
	for i, vm := range testVMs {
		testVMsInReversedOrder[len(testVMsInReversedOrder)-i-1] = vm
	}

	tests := []struct {
		name           string
		request        *v2.ListVirtualMachinesRequest
		setupMock      func(*datastoreMocks.MockDataStore)
		expectedResult *v2.ListVirtualMachinesResponse
		expectedError  string
	}{
		{
			name:    "successful list",
			request: &v2.ListVirtualMachinesRequest{},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					SearchRawVirtualMachines(ctx, gomock.Any()).
					Return(storageVMs, nil)
			},
			expectedResult: &v2.ListVirtualMachinesResponse{
				VirtualMachines: testVMs,
			},
			expectedError: "",
		},
		{
			name:    "empty list",
			request: &v2.ListVirtualMachinesRequest{},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					SearchRawVirtualMachines(ctx, gomock.Any()).
					Return([]*storage.VirtualMachine{}, nil)
			},
			expectedResult: &v2.ListVirtualMachinesResponse{
				VirtualMachines: []*v2.VirtualMachine{},
			},
			expectedError: "",
		},
		{
			name:    "datastore error",
			request: &v2.ListVirtualMachinesRequest{},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					SearchRawVirtualMachines(ctx, gomock.Any()).
					Return(nil, errors.New("datastore error"))
			},
			expectedResult: nil,
			expectedError:  "datastore error",
		},
		{
			name:    "default sort order",
			request: &v2.ListVirtualMachinesRequest{},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					SearchRawVirtualMachines(ctx, gomock.Any()).
					Return(storageVMs, nil)
			},
			expectedResult: &v2.ListVirtualMachinesResponse{
				VirtualMachines: testVMs,
			},
		},
		{
			name: "honor requested sort order",
			request: &v2.ListVirtualMachinesRequest{
				Query: &v2.RawQuery{
					Pagination: &v2.Pagination{
						SortOptions: []*v2.SortOption{
							{
								Field: search.ClusterID.String(),
							},
						},
					},
				},
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					SearchRawVirtualMachines(ctx, gomock.Any()).
					Return(storageVMsInReversedOrder, nil)
			},
			expectedResult: &v2.ListVirtualMachinesResponse{
				VirtualMachines: testVMsInReversedOrder,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDatastore := datastoreMocks.NewMockDataStore(ctrl)

			service := &serviceImpl{
				datastore: mockDatastore,
			}

			tt.setupMock(mockDatastore)

			result, err := service.ListVirtualMachines(ctx, tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				protoassert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
