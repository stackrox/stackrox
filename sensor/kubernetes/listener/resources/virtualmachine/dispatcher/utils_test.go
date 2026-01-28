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
		name        string
		vm          *sensorVirtualMachine.Info
		expected    int32
		expectedSet bool
	}{
		{
			name:        "nil input",
			vm:          nil,
			expected:    expectedZero,
			expectedSet: false,
		},
		{
			name:        "virtual machine with partial data is not assumed to have a VSock",
			vm:          &sensorVirtualMachine.Info{},
			expected:    expectedZero,
			expectedSet: false,
		},
		{
			name: "hypervisor",
			vm: &sensorVirtualMachine.Info{
				VSOCKCID: &zeroCID,
			},
			expected:    expectedZero,
			expectedSet: true,
		},
		{
			name: "virtual machine",
			vm: &sensorVirtualMachine.Info{
				VSOCKCID: &someCID,
			},
			expected:    int32(someCID),
			expectedSet: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			vSockCID, vSockCIDSet := getVirtualMachineVSockCID(tc.vm)
			assert.Equal(it, tc.expected, vSockCID)
			assert.Equal(it, tc.expectedSet, vSockCIDSet)
		})
	}
}

func TestGetFacts(t *testing.T) {
	tests := map[string]struct {
		vm       *sensorVirtualMachine.Info
		expected map[string]string
	}{
		"should include description and network facts when present": {
			vm: &sensorVirtualMachine.Info{
				GuestOS:     "Red Hat Enterprise Linux",
				Description: "test description",
				NodeName:    "node-1",
				IPAddresses: []string{"10.0.0.2", "10.0.0.1"},
				ActivePods:  []string{"pod-2=node-b", "pod-1=node-a"},
				BootOrder:   []string{"disk2=2", "disk1=1"},
				CDRomDisks:  []string{"cd2", "cd1"},
			},
			expected: map[string]string{
				GuestOSKey:     "Red Hat Enterprise Linux",
				DescriptionKey: "test description",
				NodeNameKey:    "node-1",
				IPAddressesKey: "10.0.0.2, 10.0.0.1",
				ActivePodsKey:  "pod-2=node-b, pod-1=node-a",
				BootOrderKey:   "disk2=2, disk1=1",
				CDRomDisksKey:  "cd2, cd1",
			},
		},
		"should preserve boot order sequence": {
			vm: &sensorVirtualMachine.Info{
				GuestOS:   "Red Hat Enterprise Linux",
				BootOrder: []string{"disk-b=1", "disk-a=1", "disk-c=2"},
			},
			expected: map[string]string{
				GuestOSKey:   "Red Hat Enterprise Linux",
				BootOrderKey: "disk-b=1, disk-a=1, disk-c=2",
			},
		},
		"should return unknown guest os when optional data is missing": {
			vm: &sensorVirtualMachine.Info{},
			expected: map[string]string{
				GuestOSKey: UnknownGuestOS,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			facts := getFacts(tt.vm)
			assert.Equal(it, tt.expected, facts)
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
						Id:          vm1ID,
						Namespace:   ns1,
						Name:        vm1Name,
						ClusterId:   fixtureconsts.Cluster2,
						VsockCid:    0,
						VsockCidSet: false,
						State:       virtualMachineV1.VirtualMachine_STOPPED,
						Facts:       getFactsForTest(t, UnknownGuestOS),
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
				GuestOS:   "Red Hat Enterprise Linux",
			},
			expected: &central.SensorEvent{
				Id:     vm2ID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:          vm2ID,
						Namespace:   ns1,
						Name:        vm2Name,
						ClusterId:   fixtureconsts.Cluster2,
						VsockCid:    wireVM2VSockCID,
						VsockCidSet: true,
						State:       virtualMachineV1.VirtualMachine_RUNNING,
						Facts: map[string]string{
							GuestOSKey: "Red Hat Enterprise Linux",
						},
					},
				},
			},
		},
		{
			name:      "sync stopped virtual machine",
			action:    central.ResourceAction_SYNC_RESOURCE,
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
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:          vm1ID,
						Namespace:   ns1,
						Name:        vm1Name,
						ClusterId:   fixtureconsts.Cluster2,
						VsockCid:    0,
						VsockCidSet: false,
						State:       virtualMachineV1.VirtualMachine_STOPPED,
						Facts:       getFactsForTest(t, UnknownGuestOS),
					},
				},
			},
		},
		{
			name:      "sync running virtual machine",
			action:    central.ResourceAction_SYNC_RESOURCE,
			clusterID: fixtureconsts.Cluster2,
			inputVM: &sensorVirtualMachine.Info{
				ID:        sensorVirtualMachine.VMID(vm2ID),
				Name:      vm2Name,
				Namespace: ns1,
				VSOCKCID:  &vm2VSockCID,
				Running:   true,
				GuestOS:   "Red Hat Enterprise Linux",
			},
			expected: &central.SensorEvent{
				Id:     vm2ID,
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:          vm2ID,
						Namespace:   ns1,
						Name:        vm2Name,
						ClusterId:   fixtureconsts.Cluster2,
						VsockCid:    wireVM2VSockCID,
						VsockCidSet: true,
						State:       virtualMachineV1.VirtualMachine_RUNNING,
						Facts: map[string]string{
							GuestOSKey: "Red Hat Enterprise Linux",
						},
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
				GuestOS:   "Red Hat Enterprise Linux",
			},
			expected: &central.SensorEvent{
				Id:     vm2ID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:          vm2ID,
						Namespace:   ns1,
						Name:        vm2Name,
						ClusterId:   fixtureconsts.Cluster2,
						VsockCid:    wireVM2VSockCID,
						VsockCidSet: true,
						State:       virtualMachineV1.VirtualMachine_RUNNING,
						Facts: map[string]string{
							GuestOSKey: "Red Hat Enterprise Linux",
						},
					},
				},
			},
		},
		{
			name:      "remove stopped virtual machine",
			action:    central.ResourceAction_REMOVE_RESOURCE,
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
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:          vm1ID,
						Namespace:   ns1,
						Name:        vm1Name,
						ClusterId:   fixtureconsts.Cluster2,
						VsockCid:    0,
						VsockCidSet: false,
						State:       virtualMachineV1.VirtualMachine_STOPPED,
						Facts:       getFactsForTest(t, UnknownGuestOS),
					},
				},
			},
		},
		{
			name:      "remove running virtual machine",
			action:    central.ResourceAction_REMOVE_RESOURCE,
			clusterID: fixtureconsts.Cluster2,
			inputVM: &sensorVirtualMachine.Info{
				ID:        sensorVirtualMachine.VMID(vm2ID),
				Name:      vm2Name,
				Namespace: ns1,
				VSOCKCID:  &vm2VSockCID,
				Running:   true,
				GuestOS:   "Red Hat Enterprise Linux",
			},
			expected: &central.SensorEvent{
				Id:     vm2ID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:          vm2ID,
						Namespace:   ns1,
						Name:        vm2Name,
						ClusterId:   fixtureconsts.Cluster2,
						VsockCid:    wireVM2VSockCID,
						VsockCidSet: true,
						State:       virtualMachineV1.VirtualMachine_RUNNING,
						Facts: map[string]string{
							GuestOSKey: "Red Hat Enterprise Linux",
						},
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

func getFactsForTest(_ *testing.T, guestOS string) map[string]string {
	return map[string]string{
		GuestOSKey: guestOS,
	}
}
