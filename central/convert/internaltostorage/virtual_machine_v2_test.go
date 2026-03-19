package internaltostorage

import (
	"testing"

	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stretchr/testify/assert"
)

func TestVirtualMachineV2(t *testing.T) {
	tests := []struct {
		name     string
		input    *virtualMachineV1.VirtualMachine
		expected *storage.VirtualMachineV2
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "basic virtual machine",
			input: &virtualMachineV1.VirtualMachine{
				Id:        "VM-ID-1",
				Namespace: "virtual-machine-namespace",
				Name:      "virtual-machine-name",
				ClusterId: uuid.NewTestUUID(1).String(),
				Facts: map[string]string{
					virtualmachine.GuestOSKey: "Red Hat Enterprise Linux",
				},
				VsockCid: 42,
				State:    virtualMachineV1.VirtualMachine_RUNNING,
			},
			expected: &storage.VirtualMachineV2{
				Id:        "VM-ID-1",
				Namespace: "virtual-machine-namespace",
				Name:      "virtual-machine-name",
				ClusterId: uuid.NewTestUUID(1).String(),
				Facts: map[string]string{
					virtualmachine.GuestOSKey: "Red Hat Enterprise Linux",
				},
				GuestOs:  "Red Hat Enterprise Linux",
				VsockCid: 42,
				State:    storage.VirtualMachineV2_RUNNING,
			},
		},
		{
			name: "virtual machine without guestOS fact",
			input: &virtualMachineV1.VirtualMachine{
				Id:        "VM-ID-2",
				Namespace: "ns",
				Name:      "vm-2",
				ClusterId: uuid.NewTestUUID(2).String(),
				Facts: map[string]string{
					"nodeName": "node-1",
				},
				State: virtualMachineV1.VirtualMachine_STOPPED,
			},
			expected: &storage.VirtualMachineV2{
				Id:        "VM-ID-2",
				Namespace: "ns",
				Name:      "vm-2",
				ClusterId: uuid.NewTestUUID(2).String(),
				Facts: map[string]string{
					"nodeName": "node-1",
				},
				GuestOs: "",
				State:   storage.VirtualMachineV2_STOPPED,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			converted := VirtualMachineV2(tt.input)
			protoassert.Equal(it, tt.expected, converted)
		})
	}
}

func TestConvertVirtualMachineV2State(t *testing.T) {
	tests := []struct {
		name     string
		input    virtualMachineV1.VirtualMachine_State
		expected storage.VirtualMachineV2_State
	}{
		{
			name:     "UNKNOWN",
			input:    virtualMachineV1.VirtualMachine_UNKNOWN,
			expected: storage.VirtualMachineV2_UNKNOWN,
		},
		{
			name:     "STOPPED",
			input:    virtualMachineV1.VirtualMachine_STOPPED,
			expected: storage.VirtualMachineV2_STOPPED,
		},
		{
			name:     "RUNNING",
			input:    virtualMachineV1.VirtualMachine_RUNNING,
			expected: storage.VirtualMachineV2_RUNNING,
		},
		{
			name:     "Other",
			input:    virtualMachineV1.VirtualMachine_State(-1),
			expected: storage.VirtualMachineV2_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			got := convertVirtualMachineV2State(tt.input)
			assert.Equal(it, tt.expected, got)
		})
	}
}
