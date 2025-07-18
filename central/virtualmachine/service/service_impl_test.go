package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/convert/v2tostorage"
	"github.com/stackrox/rox/central/role/sachelper/mocks"
	datastoreMocks "github.com/stackrox/rox/central/virtualmachine/datastore/mocks"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestCreateVirtualMachine(t *testing.T) {
	ctx := context.Background()

	testVM := &v2.VirtualMachine{
		Id:        "test-vm-id",
		Name:      "test-vm",
		Namespace: "test-namespace",
	}

	tests := []struct {
		name           string
		request        *v2.CreateVirtualMachineRequest
		setupMock      func(*datastoreMocks.MockDataStore)
		expectedResult *v2.VirtualMachine
		expectedError  string
	}{
		{
			name: "successful creation",
			request: &v2.CreateVirtualMachineRequest{
				VirtualMachine: testVM,
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					CreateVirtualMachine(ctx, v2tostorage.VirtualMachine(testVM)).
					Return(nil)
			},
			expectedResult: testVM,
			expectedError:  "",
		},
		{
			name:           "nil request",
			request:        nil,
			setupMock:      func(mockDS *datastoreMocks.MockDataStore) {},
			expectedResult: nil,
			expectedError:  "id must be specified",
		},
		{
			name: "empty ID",
			request: &v2.CreateVirtualMachineRequest{
				VirtualMachine: &v2.VirtualMachine{
					Id: "",
				},
			},
			setupMock:      func(mockDS *datastoreMocks.MockDataStore) {},
			expectedResult: nil,
			expectedError:  "id must be specified",
		},
		{
			name: "datastore error",
			request: &v2.CreateVirtualMachineRequest{
				VirtualMachine: testVM,
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					CreateVirtualMachine(ctx, v2tostorage.VirtualMachine(testVM)).
					Return(errors.New("datastore error"))
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
			mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

			service := &serviceImpl{
				datastore:        mockDatastore,
				clusterSACHelper: mockSACHelper,
			}

			tt.setupMock(mockDatastore)

			result, err := service.CreateVirtualMachine(ctx, tt.request)

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

func TestGetVirtualMachine(t *testing.T) {
	ctx := context.Background()

	testVM := &v2.VirtualMachine{
		Id:        "test-vm-id",
		Name:      "test-vm",
		Namespace: "test-namespace",
	}

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
					Return(v2tostorage.VirtualMachine(testVM), true, nil)
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
			mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

			service := &serviceImpl{
				datastore:        mockDatastore,
				clusterSACHelper: mockSACHelper,
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

	testVMs := []*v2.VirtualMachine{
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

	storageVMs := make([]*storage.VirtualMachine, len(testVMs))

	for i, vm := range testVMs {
		storageVMs[i] = v2tostorage.VirtualMachine(vm)
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
					GetAllVirtualMachines(ctx).
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
					GetAllVirtualMachines(ctx).
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
					GetAllVirtualMachines(ctx).
					Return(nil, errors.New("datastore error"))
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
			mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

			service := &serviceImpl{
				datastore:        mockDatastore,
				clusterSACHelper: mockSACHelper,
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

func TestDeleteVirtualMachine(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		request        *v2.DeleteVirtualMachineRequest
		setupMock      func(*datastoreMocks.MockDataStore)
		expectedResult *v2.DeleteVirtualMachineResponse
		expectedError  string
	}{
		{
			name: "successful deletion",
			request: &v2.DeleteVirtualMachineRequest{
				Id: "test-vm-id",
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					DeleteVirtualMachines(ctx, "test-vm-id").
					Return(nil)
			},
			expectedResult: &v2.DeleteVirtualMachineResponse{
				Success: true,
			},
			expectedError: "",
		},
		{
			name: "empty ID",
			request: &v2.DeleteVirtualMachineRequest{
				Id: "",
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {},
			expectedResult: &v2.DeleteVirtualMachineResponse{
				Success: false,
			},
			expectedError: "id cannot be empty",
		},
		{
			name: "datastore error",
			request: &v2.DeleteVirtualMachineRequest{
				Id: "test-vm-id",
			},
			setupMock: func(mockDS *datastoreMocks.MockDataStore) {
				mockDS.EXPECT().
					DeleteVirtualMachines(ctx, "test-vm-id").
					Return(errors.New("datastore error"))
			},
			expectedResult: &v2.DeleteVirtualMachineResponse{
				Success: false,
			},
			expectedError: "datastore error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
			mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

			service := &serviceImpl{
				datastore:        mockDatastore,
				clusterSACHelper: mockSACHelper,
			}

			tt.setupMock(mockDatastore)

			result, err := service.DeleteVirtualMachine(ctx, tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				protoassert.Equal(t, tt.expectedResult, result)
			} else {
				require.NoError(t, err)
				protoassert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
