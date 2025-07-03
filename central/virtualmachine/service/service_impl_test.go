package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/role/sachelper/mocks"
	datastoreMocks "github.com/stackrox/rox/central/virtualmachine/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestCreateVirtualMachine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
	mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

	service := &serviceImpl{
		datastore:        mockDatastore,
		clusterSACHelper: mockSACHelper,
	}

	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		vm := &storage.VirtualMachine{
			Id:        "test-vm-id",
			Name:      "test-vm",
			Namespace: "test-namespace",
		}

		request := &v1.CreateVirtualMachineRequest{
			VirtualMachine: vm,
		}

		mockDatastore.EXPECT().
			UpsertVirtualMachine(ctx, vm).
			Return(nil)

		result, err := service.CreateVirtualMachine(ctx, request)
		require.NoError(t, err)
		assert.Equal(t, vm, result)
	})

	t.Run("nil request", func(t *testing.T) {
		result, err := service.CreateVirtualMachine(ctx, nil)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id must be specified")
	})

	t.Run("empty ID", func(t *testing.T) {
		request := &v1.CreateVirtualMachineRequest{
			VirtualMachine: &storage.VirtualMachine{
				Id: "",
			},
		}

		result, err := service.CreateVirtualMachine(ctx, request)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id must be specified")
	})

	t.Run("datastore error", func(t *testing.T) {
		vm := &storage.VirtualMachine{
			Id:        "test-vm-id",
			Name:      "test-vm",
			Namespace: "test-namespace",
		}

		request := &v1.CreateVirtualMachineRequest{
			VirtualMachine: vm,
		}

		expectedErr := errors.New("datastore error")
		mockDatastore.EXPECT().
			UpsertVirtualMachine(ctx, vm).
			Return(expectedErr)

		result, err := service.CreateVirtualMachine(ctx, request)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
	})
}

func TestGetVirtualMachine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
	mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

	service := &serviceImpl{
		datastore:        mockDatastore,
		clusterSACHelper: mockSACHelper,
	}

	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		vmID := "test-vm-id"
		vm := &storage.VirtualMachine{
			Id:        vmID,
			Name:      "test-vm",
			Namespace: "test-namespace",
		}

		request := &v1.GetVirtualMachineRequest{
			Id: vmID,
		}

		mockDatastore.EXPECT().
			GetVirtualMachine(ctx, vmID).
			Return(vm, true, nil)

		result, err := service.GetVirtualMachine(ctx, request)
		require.NoError(t, err)
		assert.Equal(t, vm, result)
	})

	t.Run("empty ID", func(t *testing.T) {
		request := &v1.GetVirtualMachineRequest{
			Id: "",
		}

		result, err := service.GetVirtualMachine(ctx, request)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id must be specified")
	})

	t.Run("virtual machine not found", func(t *testing.T) {
		vmID := "non-existent-vm"
		request := &v1.GetVirtualMachineRequest{
			Id: vmID,
		}

		mockDatastore.EXPECT().
			GetVirtualMachine(ctx, vmID).
			Return(nil, false, nil)

		result, err := service.GetVirtualMachine(ctx, request)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("datastore error", func(t *testing.T) {
		vmID := "test-vm-id"
		request := &v1.GetVirtualMachineRequest{
			Id: vmID,
		}

		expectedErr := errors.New("datastore error")
		mockDatastore.EXPECT().
			GetVirtualMachine(ctx, vmID).
			Return(nil, false, expectedErr)

		result, err := service.GetVirtualMachine(ctx, request)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
	})
}

func TestListVirtualMachines(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}
		vms := []*storage.VirtualMachine{
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

		request := &v1.RawQuery{
			Query: "name:test",
		}

		mockDatastore.EXPECT().
			SearchRawVirtualMachines(ctx, gomock.Any()).
			Return(vms, nil)

		result, err := service.ListVirtualMachines(ctx, request)
		require.NoError(t, err)
		assert.Equal(t, vms, result.VirtualMachines)
	})

	t.Run("empty query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		vms := []*storage.VirtualMachine{}

		request := &v1.RawQuery{
			Query: "",
		}

		mockDatastore.EXPECT().
			SearchRawVirtualMachines(ctx, gomock.Any()).
			Return(vms, nil)

		result, err := service.ListVirtualMachines(ctx, request)
		require.NoError(t, err)
		assert.Empty(t, result.VirtualMachines)
	})

	t.Run("invalid query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		request := &v1.RawQuery{
			Query: "invalid[query",
		}

		// The query parser is lenient, so this will reach the datastore
		mockDatastore.EXPECT().
			SearchRawVirtualMachines(ctx, gomock.Any()).
			Return(nil, errors.New("mock datastore error"))

		result, err := service.ListVirtualMachines(ctx, request)
		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("datastore error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		request := &v1.RawQuery{
			Query: "name:test",
		}

		expectedErr := errors.New("datastore error")
		mockDatastore.EXPECT().
			SearchRawVirtualMachines(ctx, gomock.Any()).
			Return(nil, expectedErr)

		result, err := service.ListVirtualMachines(ctx, request)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
	})
}

func TestDeleteVirtualMachine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
	mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

	service := &serviceImpl{
		datastore:        mockDatastore,
		clusterSACHelper: mockSACHelper,
	}

	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		vmID := "test-vm-id"
		request := &v1.DeleteVirtualMachineRequest{
			Id: vmID,
		}

		mockDatastore.EXPECT().
			DeleteVirtualMachines(ctx, vmID).
			Return(nil)

		result, err := service.DeleteVirtualMachine(ctx, request)
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("empty ID", func(t *testing.T) {
		request := &v1.DeleteVirtualMachineRequest{
			Id: "",
		}

		result, err := service.DeleteVirtualMachine(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id cannot be empty")
		assert.False(t, result.Success)
	})

	t.Run("datastore error", func(t *testing.T) {
		vmID := "test-vm-id"
		request := &v1.DeleteVirtualMachineRequest{
			Id: vmID,
		}

		expectedErr := errors.New("datastore error")
		mockDatastore.EXPECT().
			DeleteVirtualMachines(ctx, vmID).
			Return(expectedErr)

		result, err := service.DeleteVirtualMachine(ctx, request)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result.Success)
	})
}

func TestDeleteVirtualMachines(t *testing.T) {
	ctx := context.Background()

	t.Run("successful bulk deletion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}
		searchResults := []search.Result{
			{ID: "vm-1"},
			{ID: "vm-2"},
		}

		request := &v1.DeleteVirtualMachinesRequest{
			Query: &v1.RawQuery{
				Query: "cluster:test",
			},
			Confirm: true,
		}

		mockDatastore.EXPECT().
			Search(ctx, gomock.Any()).
			Return(searchResults, nil)

		mockDatastore.EXPECT().
			DeleteVirtualMachines(ctx, "vm-1", "vm-2").
			Return(nil)

		result, err := service.DeleteVirtualMachines(ctx, request)
		require.NoError(t, err)
		assert.Equal(t, uint32(2), result.NumDeleted)
		assert.False(t, result.DryRun)
	})

	t.Run("dry run", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		searchResults := []search.Result{
			{ID: "vm-1"},
			{ID: "vm-2"},
		}

		request := &v1.DeleteVirtualMachinesRequest{
			Query: &v1.RawQuery{
				Query: "cluster:test",
			},
			Confirm: false,
		}

		mockDatastore.EXPECT().
			Search(ctx, gomock.Any()).
			Return(searchResults, nil)

		// No deletion should occur in dry run
		result, err := service.DeleteVirtualMachines(ctx, request)
		require.NoError(t, err)
		assert.Equal(t, uint32(2), result.NumDeleted)
		assert.True(t, result.DryRun)
	})

	t.Run("nil query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		request := &v1.DeleteVirtualMachinesRequest{
			Query:   nil,
			Confirm: true,
		}

		result, err := service.DeleteVirtualMachines(ctx, request)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scoping query is required")
	})

	t.Run("invalid query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		request := &v1.DeleteVirtualMachinesRequest{
			Query: &v1.RawQuery{
				Query: "invalid[query",
			},
			Confirm: true,
		}

		// The query parser is lenient, so this will reach the datastore
		mockDatastore.EXPECT().
			Search(ctx, gomock.Any()).
			Return(nil, errors.New("mock datastore error"))

		result, err := service.DeleteVirtualMachines(ctx, request)
		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("search error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		request := &v1.DeleteVirtualMachinesRequest{
			Query: &v1.RawQuery{
				Query: "cluster:test",
			},
			Confirm: true,
		}

		expectedErr := errors.New("search error")
		mockDatastore.EXPECT().
			Search(ctx, gomock.Any()).
			Return(nil, expectedErr)

		result, err := service.DeleteVirtualMachines(ctx, request)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("deletion error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDatastore := datastoreMocks.NewMockDataStore(ctrl)
		mockSACHelper := mocks.NewMockClusterSacHelper(ctrl)

		service := &serviceImpl{
			datastore:        mockDatastore,
			clusterSACHelper: mockSACHelper,
		}

		searchResults := []search.Result{
			{ID: "vm-1"},
		}

		request := &v1.DeleteVirtualMachinesRequest{
			Query: &v1.RawQuery{
				Query: "cluster:test",
			},
			Confirm: true,
		}

		mockDatastore.EXPECT().
			Search(ctx, gomock.Any()).
			Return(searchResults, nil)

		expectedErr := errors.New("deletion error")
		mockDatastore.EXPECT().
			DeleteVirtualMachines(ctx, "vm-1").
			Return(expectedErr)

		result, err := service.DeleteVirtualMachines(ctx, request)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
	})
}