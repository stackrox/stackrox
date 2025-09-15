package v2tostorage

import (
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVirtualMachine(t *testing.T) {
	timestamp := timestamppb.New(time.Now())

	tests := []struct {
		name     string
		input    *v2.VirtualMachine
		expected *storage.VirtualMachine
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete virtual machine",
			input: &v2.VirtualMachine{
				Id:          "vm-123",
				Namespace:   "default",
				Name:        "test-vm",
				ClusterId:   "cluster-456",
				ClusterName: "test-cluster",
				VsockCid:    int32(81),
				State:       v2.VirtualMachine_RUNNING,
				LastUpdated: timestamp,
			},
			expected: &storage.VirtualMachine{
				Id:          "vm-123",
				Namespace:   "default",
				Name:        "test-vm",
				ClusterId:   "cluster-456",
				ClusterName: "test-cluster",
				VsockCid:    int32(81),
				State:       storage.VirtualMachine_RUNNING,
				LastUpdated: timestamp,
			},
		},
		{
			name: "stopped virtual machine",
			input: &v2.VirtualMachine{
				Id:        "vm-stopped",
				Namespace: "test",
				Name:      "stopped-vm",
				State:     v2.VirtualMachine_STOPPED,
			},
			expected: &storage.VirtualMachine{
				Id:        "vm-stopped",
				Namespace: "test",
				Name:      "stopped-vm",
				State:     storage.VirtualMachine_STOPPED,
			},
		},
		{
			name: "minimal virtual machine",
			input: &v2.VirtualMachine{
				Id:        "vm-minimal",
				Namespace: "test",
				Name:      "minimal-vm",
			},
			expected: &storage.VirtualMachine{
				Id:        "vm-minimal",
				Namespace: "test",
				Name:      "minimal-vm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VirtualMachine(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertVirtualMachineState(t *testing.T) {
	tests := []struct {
		name     string
		input    v2.VirtualMachine_State
		expected storage.VirtualMachine_State
	}{
		{
			name:     "UNKNOWN",
			input:    v2.VirtualMachine_UNKNOWN,
			expected: storage.VirtualMachine_UNKNOWN,
		},
		{
			name:     "STOPPED",
			input:    v2.VirtualMachine_STOPPED,
			expected: storage.VirtualMachine_STOPPED,
		},
		{
			name:     "RUNNING",
			input:    v2.VirtualMachine_RUNNING,
			expected: storage.VirtualMachine_RUNNING,
		},
		{
			name:     "Other",
			input:    v2.VirtualMachine_State(-1),
			expected: storage.VirtualMachine_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVirtualMachineState(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
