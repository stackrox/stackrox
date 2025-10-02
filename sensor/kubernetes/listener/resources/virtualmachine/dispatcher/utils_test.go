package dispatcher

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	sensorVirtualMachine "github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stretchr/testify/assert"
)

func TestGetVirtualMachineState(t *testing.T) {
	tests := []struct {
		name     string
		vm       *sensorVirtualMachine.Info
		expected virtualMachineV1.VirtualMachine_State
	}{
		{
			name:     "nil input",
			vm:       nil,
			expected: virtualMachineV1.VirtualMachine_UNKNOWN,
		},
		{
			name: "running machine",
			vm: &sensorVirtualMachine.Info{
				Running: true,
			},
			expected: virtualMachineV1.VirtualMachine_RUNNING,
		},
		{
			name: "stopped machine",
			vm: &sensorVirtualMachine.Info{
				Running: false,
			},
			expected: virtualMachineV1.VirtualMachine_STOPPED,
		},
		{
			name:     "machine with partial data is assumed not running",
			vm:       &sensorVirtualMachine.Info{},
			expected: virtualMachineV1.VirtualMachine_STOPPED,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			state := getVirtualMachineState(tc.vm)
			assert.Equal(it, tc.expected, state)
		})
	}
}

func TestGetVirtualMachineVSockCID(t *testing.T) {
	const expectedZero = int32(0)
	zeroCID := uint32(0)
	someCID := uint32(0xca7d09)
	tests := []struct {
		name     string
		vm       *sensorVirtualMachine.Info
		expected int32
	}{
		{
			name:     "nil input",
			vm:       nil,
			expected: expectedZero,
		},
		{
			name:     "virtual machine with partial data is not assumed to have a VSock",
			vm:       &sensorVirtualMachine.Info{},
			expected: expectedZero,
		},
		{
			name: "hypervisor",
			vm: &sensorVirtualMachine.Info{
				VSOCKCID: &zeroCID,
			},
			expected: expectedZero,
		},
		{
			name: "virtual machine",
			vm: &sensorVirtualMachine.Info{
				VSOCKCID: &someCID,
			},
			expected: int32(someCID),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			state := getVirtualMachineVSockCID(tc.vm)
			assert.Equal(it, tc.expected, state)
		})
	}
}

func TestCreateEvent(t *testing.T) {
	const ns1 = "namespace-1"
	var vm1ID = uuid.NewTestUUID(1).String()
	const vm1Name = "Test VM 1"
	var vm2ID = uuid.NewTestUUID(2).String()
	const vm2Name = "Test VM 2"
	vm2VSockCID := uint32(0xd09ca7)
	wireVM2VSockCID := int32(vm2VSockCID)

	tests := []struct {
		name      string
		action    central.ResourceAction
		clusterID string
		inputVM   *sensorVirtualMachine.Info
		expected  *central.SensorEvent
	}{
		{
			name:      "nil input",
			action:    central.ResourceAction_UPDATE_RESOURCE,
			clusterID: fixtureconsts.Cluster1,
			inputVM:   nil,
			expected:  nil,
		},
		{
			name:      "create stopped virtual machine",
			action:    central.ResourceAction_CREATE_RESOURCE,
			clusterID: fixtureconsts.Cluster2,
			inputVM: &sensorVirtualMachine.Info{
				ID:        sensorVirtualMachine.VMID(vm1ID),
				Name:      vm1Name,
				Namespace: ns1,
				VSOCKCID:  nil,
				Running:   false,
			},
			expected: &central.SensorEvent{
				Id:     vm1ID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vm1ID,
						Namespace: ns1,
						Name:      vm1Name,
						ClusterId: fixtureconsts.Cluster2,
						VsockCid:  0,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
					},
				},
			},
		},
		{
			name:      "create running virtual machine",
			action:    central.ResourceAction_CREATE_RESOURCE,
			clusterID: fixtureconsts.Cluster2,
			inputVM: &sensorVirtualMachine.Info{
				ID:        sensorVirtualMachine.VMID(vm2ID),
				Name:      vm2Name,
				Namespace: ns1,
				VSOCKCID:  &vm2VSockCID,
				Running:   true,
			},
			expected: &central.SensorEvent{
				Id:     vm2ID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vm2ID,
						Namespace: ns1,
						Name:      vm2Name,
						ClusterId: fixtureconsts.Cluster2,
						VsockCid:  wireVM2VSockCID,
						State:     virtualMachineV1.VirtualMachine_RUNNING,
					},
				},
			},
		},
		{
			name:      "update running virtual machine",
			action:    central.ResourceAction_UPDATE_RESOURCE,
			clusterID: fixtureconsts.Cluster2,
			inputVM: &sensorVirtualMachine.Info{
				ID:        sensorVirtualMachine.VMID(vm2ID),
				Name:      vm2Name,
				Namespace: ns1,
				VSOCKCID:  &vm2VSockCID,
				Running:   true,
			},
			expected: &central.SensorEvent{
				Id:     vm2ID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vm2ID,
						Namespace: ns1,
						Name:      vm2Name,
						ClusterId: fixtureconsts.Cluster2,
						VsockCid:  wireVM2VSockCID,
						State:     virtualMachineV1.VirtualMachine_RUNNING,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			event := createEvent(tt.action, tt.clusterID, tt.inputVM)
			protoassert.Equal(it, tt.expected, event)
		})
	}
}
