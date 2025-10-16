package internaltostorage

import (
	"testing"

	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestVirtualMachine(t *testing.T) {
	vm := &virtualMachineV1.VirtualMachine{}
	vm.SetId("VM-ID-1")
	vm.SetNamespace("virtual-machine-namespace")
	vm.SetName("virtual-machine-name")
	vm.SetClusterId(uuid.NewTestUUID(1).String())
	vm.SetVsockCid(42)
	vm.SetState(virtualMachineV1.VirtualMachine_RUNNING)
	vm2 := &storage.VirtualMachine{}
	vm2.SetId("VM-ID-1")
	vm2.SetNamespace("virtual-machine-namespace")
	vm2.SetName("virtual-machine-name")
	vm2.SetClusterId(uuid.NewTestUUID(1).String())
	vm2.SetVsockCid(42)
	vm2.SetState(storage.VirtualMachine_RUNNING)
	tests := []struct {
		name     string
		input    *virtualMachineV1.VirtualMachine
		expected *storage.VirtualMachine
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "basic virtual machine",
			input:    vm,
			expected: vm2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			converted := VirtualMachine(tt.input)
			protoassert.Equal(it, tt.expected, converted)
		})
	}
}

func TestConvertVirtualMachineState(t *testing.T) {
	tests := []struct {
		name     string
		input    virtualMachineV1.VirtualMachine_State
		expected storage.VirtualMachine_State
	}{
		{
			name:     "UNKNOWN",
			input:    virtualMachineV1.VirtualMachine_UNKNOWN,
			expected: storage.VirtualMachine_UNKNOWN,
		},
		{
			name:     "STOPPED",
			input:    virtualMachineV1.VirtualMachine_STOPPED,
			expected: storage.VirtualMachine_STOPPED,
		},
		{
			name:     "RUNNING",
			input:    virtualMachineV1.VirtualMachine_RUNNING,
			expected: storage.VirtualMachine_RUNNING,
		},
		{
			name:     "Other",
			input:    virtualMachineV1.VirtualMachine_State(-1),
			expected: storage.VirtualMachine_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(it *testing.T) {
			got := convertVirtualMachineState(tt.input)
			assert.Equal(it, tt.expected, got)
		})
	}
}
