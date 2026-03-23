package virtualmachines

import (
	"testing"

	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/convert/internaltostorage"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	virtualMachineDSMocks "github.com/stackrox/rox/central/virtualmachine/datastore/mocks"
	vmV2DataStoreMocks "github.com/stackrox/rox/central/virtualmachine/v2/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCapabilities(t *testing.T) {
	pipeline := &pipelineImpl{}
	assert.ElementsMatch(
		t,
		[]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported},
		pipeline.Capabilities(),
	)
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name  string
		input *central.MsgFromSensor
		want  bool
	}{
		{
			name:  "nil input",
			input: nil,
			want:  false,
		},
		{
			name:  "empty input",
			input: &central.MsgFromSensor{},
			want:  false,
		},
		{
			name: "bad message type",
			input: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Resource: &central.SensorEvent_Node{
							Node: &storage.Node{
								Id: "node1",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "match",
			input: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Resource: &central.SensorEvent_VirtualMachine{
							VirtualMachine: &virtualMachineV1.VirtualMachine{
								Id: "virtualMachine1",
							},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			pipeline := &pipelineImpl{}
			got := pipeline.Match(tt.input)
			assert.Equal(it, tt.want, got)
		})
	}
}

func TestPipelineRun(t *testing.T) {
	testClusterID := fixtureconsts.Cluster1
	type mocks struct {
		clusters        *clusterDSMocks.MockDataStore
		virtualMachines *virtualMachineDSMocks.MockDataStore
	}
	var upsertTestVM = &virtualMachineV1.VirtualMachine{
		Id:        uuid.NewTestUUID(1).String(),
		Namespace: "test-namespace",
		Name:      "test-virtual-machine",
		ClusterId: testClusterID,
		VsockCid:  0,
		State:     virtualMachineV1.VirtualMachine_STOPPED,
	}
	tests := []struct {
		name             string
		setupMocks       func(*mocks)
		message          *central.MsgFromSensor
		expectsError     bool
		expectedErrorMsg string
	}{
		{
			name:             "nil input",
			expectsError:     true,
			expectedErrorMsg: "unexpected resource type <nil> for virtual machine",
		},
		{
			name:             "bad input type",
			expectsError:     true,
			message:          getNodeMessage(),
			expectedErrorMsg: "unexpected resource type *central.SensorEvent_Node for virtual machine",
		},
		{
			name: "Removal expects call to datastore remove and succeeds",
			setupMocks: func(testMocks *mocks) {
				testMocks.virtualMachines.EXPECT().
					DeleteVirtualMachines(gomock.Any(), "removed_vm_id").
					Return(nil)
			},
			message: getVirtualMachineRemovalMessage("removed_vm_id"),
		},
		{
			name: "Addition upserts despite cluster name lookup failure",
			setupMocks: func(testMock *mocks) {
				storedVM := internaltostorage.VirtualMachine(upsertTestVM)
				storedVM.ClusterName = ""
				testMock.clusters.EXPECT().
					GetClusterName(gomock.Any(), testClusterID).
					Return("", false, nil)
				testMock.virtualMachines.EXPECT().
					UpsertVirtualMachine(gomock.Any(), protomock.GoMockMatcherEqualMessage(storedVM)).
					Return(nil)
			},
			message: getVirtualMachineAdditionMessage(upsertTestVM),
		},
		{
			name: "Addition upserts with cluster name on lookup success",
			setupMocks: func(testMock *mocks) {
				storedVM := internaltostorage.VirtualMachine(upsertTestVM)
				storedVM.ClusterName = "test-cluster"
				testMock.clusters.EXPECT().
					GetClusterName(gomock.Any(), testClusterID).
					Return("test-cluster", true, nil)
				testMock.virtualMachines.EXPECT().
					UpsertVirtualMachine(gomock.Any(), protomock.GoMockMatcherEqualMessage(storedVM)).
					Return(nil)
			},
			message: getVirtualMachineAdditionMessage(upsertTestVM),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()
			testMocks := &mocks{
				clusters:        clusterDSMocks.NewMockDataStore(mockCtrl),
				virtualMachines: virtualMachineDSMocks.NewMockDataStore(mockCtrl),
			}
			pipeline := newPipeline(testMocks.clusters, testMocks.virtualMachines, nil)
			if tt.setupMocks != nil {
				tt.setupMocks(testMocks)
			}
			err := pipeline.Run(it.Context(), testClusterID, tt.message, nil)
			if tt.expectsError {
				assert.ErrorContains(it, err, tt.expectedErrorMsg)
			} else {
				assert.NoError(it, err)
			}
		})
	}
}

func TestPipelineReconcile(t *testing.T) {
	testClusterID := fixtureconsts.Cluster1
	otherClusterID := fixtureconsts.Cluster2
	tests := []struct {
		name          string
		setupStoreMap func(*reconciliation.StoreMap)
		setupMock     func(*virtualMachineDSMocks.MockDataStore)
		expectsError  bool
	}{
		{
			name: "reconciliation has nothing to remove",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *virtualMachineDSMocks.MockDataStore) {
				m.EXPECT().SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return([]*storage.VirtualMachine{
						{
							Id:        "existing-vm",
							ClusterId: testClusterID,
						},
					}, nil)
			},
		},
		{
			name: "reconciliation does not remove virtual machines from other clusters",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *virtualMachineDSMocks.MockDataStore) {
				m.EXPECT().SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return([]*storage.VirtualMachine{
						{
							Id:        "existing-vm",
							ClusterId: testClusterID,
						},
						{
							Id:        "existing-vm-in-other-cluster",
							ClusterId: otherClusterID,
						},
					}, nil)
			},
		},
		{
			name: "reconciliation does not remove virtual machines from other clusters",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *virtualMachineDSMocks.MockDataStore) {
				m.EXPECT().
					SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return([]*storage.VirtualMachine{
						{
							Id:        "existing-vm",
							ClusterId: testClusterID,
						},
						{
							Id:        "vm-to-remove-from-cluster",
							ClusterId: testClusterID,
						},
					}, nil)
				m.EXPECT().
					DeleteVirtualMachines(gomock.Any(), "vm-to-remove-from-cluster").
					Return(nil)
			},
		},
		{
			name: "reconciliation fails on virtual machine lookup error",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *virtualMachineDSMocks.MockDataStore) {
				m.EXPECT().
					SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return(nil, errox.InvalidArgs)
			},
			expectsError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()

			mockVMStore := virtualMachineDSMocks.NewMockDataStore(mockCtrl)
			if tt.setupMock != nil {
				tt.setupMock(mockVMStore)
			}

			storeMap := reconciliation.NewStoreMap()
			if tt.setupStoreMap != nil {
				tt.setupStoreMap(storeMap)
			}

			pipeline := newPipeline(nil, mockVMStore, nil)
			err := pipeline.Reconcile(it.Context(), testClusterID, storeMap)
			if !tt.expectsError {
				assert.NoError(it, err)
			} else {
				assert.Error(it, err)
			}
		})
	}
}

func getNodeMessage() *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Node{},
			},
		},
	}
}

func getVirtualMachineRemovalMessage(vmID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id: vmID,
					},
				},
			},
		},
	}
}

func getVirtualMachineAdditionMessage(vm *virtualMachineV1.VirtualMachine) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: vm,
				},
			},
		},
	}
}

func TestPipelineRunV2(t *testing.T) {
	testClusterID := fixtureconsts.Cluster1
	type v2Mocks struct {
		clusters *clusterDSMocks.MockDataStore
		vmV2     *vmV2DataStoreMocks.MockDataStore
	}
	var upsertTestVM = &virtualMachineV1.VirtualMachine{
		Id:        uuid.NewTestUUID(1).String(),
		Namespace: "test-namespace",
		Name:      "test-virtual-machine",
		ClusterId: testClusterID,
		VsockCid:  0,
		State:     virtualMachineV1.VirtualMachine_STOPPED,
	}
	tests := []struct {
		name             string
		setupMocks       func(*v2Mocks)
		message          *central.MsgFromSensor
		expectsError     bool
		expectedErrorMsg string
	}{
		{
			name: "V2 removal calls v2 DeleteVirtualMachines",
			setupMocks: func(m *v2Mocks) {
				m.vmV2.EXPECT().
					DeleteVirtualMachines(gomock.Any(), "removed_vm_id").
					Return(nil)
			},
			message: getVirtualMachineRemovalMessage("removed_vm_id"),
		},
		{
			name: "V2 upsert with cluster name",
			setupMocks: func(m *v2Mocks) {
				storedVM := internaltostorage.VirtualMachineV2(upsertTestVM)
				storedVM.ClusterName = "test-cluster"
				m.clusters.EXPECT().
					GetClusterName(gomock.Any(), testClusterID).
					Return("test-cluster", true, nil)
				m.vmV2.EXPECT().
					UpsertVirtualMachine(gomock.Any(), protomock.GoMockMatcherEqualMessage(storedVM)).
					Return(nil)
			},
			message: getVirtualMachineAdditionMessage(upsertTestVM),
		},
		{
			name: "V2 upsert without cluster name on lookup failure",
			setupMocks: func(m *v2Mocks) {
				storedVM := internaltostorage.VirtualMachineV2(upsertTestVM)
				storedVM.ClusterName = ""
				m.clusters.EXPECT().
					GetClusterName(gomock.Any(), testClusterID).
					Return("", false, nil)
				m.vmV2.EXPECT().
					UpsertVirtualMachine(gomock.Any(), protomock.GoMockMatcherEqualMessage(storedVM)).
					Return(nil)
			},
			message: getVirtualMachineAdditionMessage(upsertTestVM),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()
			m := &v2Mocks{
				clusters: clusterDSMocks.NewMockDataStore(mockCtrl),
				vmV2:     vmV2DataStoreMocks.NewMockDataStore(mockCtrl),
			}
			pipeline := newPipeline(m.clusters, nil, m.vmV2)
			if tt.setupMocks != nil {
				tt.setupMocks(m)
			}
			err := pipeline.Run(it.Context(), testClusterID, tt.message, nil)
			if tt.expectsError {
				assert.ErrorContains(it, err, tt.expectedErrorMsg)
			} else {
				assert.NoError(it, err)
			}
		})
	}
}

func TestPipelineReconcileV2(t *testing.T) {
	testClusterID := fixtureconsts.Cluster1
	tests := []struct {
		name          string
		setupStoreMap func(*reconciliation.StoreMap)
		setupMock     func(*vmV2DataStoreMocks.MockDataStore)
		expectsError  bool
	}{
		{
			name: "v2 reconciliation has nothing to remove",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *vmV2DataStoreMocks.MockDataStore) {
				m.EXPECT().SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return([]*storage.VirtualMachineV2{
						{
							Id:        "existing-vm",
							ClusterId: testClusterID,
						},
					}, nil)
			},
		},
		{
			name: "v2 reconciliation query filters by cluster so other-cluster VMs are not returned",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *vmV2DataStoreMocks.MockDataStore) {
				// The store query is filtered by clusterID, so only matching VMs are returned.
				m.EXPECT().SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return([]*storage.VirtualMachineV2{
						{
							Id:        "existing-vm",
							ClusterId: testClusterID,
						},
					}, nil)
			},
		},
		{
			name: "v2 reconciliation removes stale VMs from cluster",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *vmV2DataStoreMocks.MockDataStore) {
				m.EXPECT().
					SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return([]*storage.VirtualMachineV2{
						{
							Id:        "existing-vm",
							ClusterId: testClusterID,
						},
						{
							Id:        "vm-to-remove-from-cluster",
							ClusterId: testClusterID,
						},
					}, nil)
				m.EXPECT().
					DeleteVirtualMachines(gomock.Any(), "vm-to-remove-from-cluster").
					Return(nil)
			},
		},
		{
			name: "v2 reconciliation fails on lookup error",
			setupStoreMap: func(m *reconciliation.StoreMap) {
				m.Add((*central.SensorEvent_VirtualMachine)(nil), "existing-vm")
			},
			setupMock: func(m *vmV2DataStoreMocks.MockDataStore) {
				m.EXPECT().
					SearchRawVirtualMachines(gomock.Any(), gomock.Any()).
					Return(nil, errox.InvalidArgs)
			},
			expectsError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()

			mockVMV2Store := vmV2DataStoreMocks.NewMockDataStore(mockCtrl)
			if tt.setupMock != nil {
				tt.setupMock(mockVMV2Store)
			}

			storeMap := reconciliation.NewStoreMap()
			if tt.setupStoreMap != nil {
				tt.setupStoreMap(storeMap)
			}

			pipeline := newPipeline(nil, nil, mockVMV2Store)
			err := pipeline.Reconcile(it.Context(), testClusterID, storeMap)
			if !tt.expectsError {
				assert.NoError(it, err)
			} else {
				assert.Error(it, err)
			}
		})
	}
}
