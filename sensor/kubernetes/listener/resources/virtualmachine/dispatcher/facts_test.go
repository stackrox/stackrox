package dispatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

func TestDescriptionFromAnnotations(t *testing.T) {
	tests := map[string]struct {
		annotations map[string]string
		expected    string
	}{
		"should return empty string when annotations are nil": {
			annotations: nil,
			expected:    "",
		},
		"should return empty string when no description keys are set": {
			annotations: map[string]string{
				"other": "value",
			},
			expected: "",
		},
		"should ignore empty values and return next available description": {
			annotations: map[string]string{
				"description":              "",
				"openshift.io/description": "openshift description",
			},
			expected: "openshift description",
		},
		"should honor description key order when multiple values exist": {
			annotations: map[string]string{
				"kubevirt.io/description":  "kubevirt description",
				"openshift.io/description": "openshift description",
				"description":              "plain description",
			},
			expected: "plain description; openshift description; kubevirt description",
		},
		"should return single description when exactly one is set": {
			annotations: map[string]string{
				"openshift.io/description": "openshift description",
				"other":                    "value",
			},
			expected: "openshift description",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tt.expected, descriptionFromAnnotations(tt.annotations))
		})
	}
}

func TestExtractIPAddresses(t *testing.T) {
	tests := map[string]struct {
		vmi      *kubeVirtV1.VirtualMachineInstance
		expected []string
	}{
		"should return nil for nil vmi": {
			vmi:      nil,
			expected: nil,
		},
		"should return nil for empty interface data": {
			vmi: &kubeVirtV1.VirtualMachineInstance{},
		},
		"should use IPs when present and deduplicate": {
			vmi: &kubeVirtV1.VirtualMachineInstance{
				Status: kubeVirtV1.VirtualMachineInstanceStatus{
					Interfaces: []kubeVirtV1.VirtualMachineInstanceNetworkInterface{
						{IPs: []string{"10.0.0.2", "10.0.0.1", "10.0.0.2"}, IP: "10.0.0.9"},
						{IPs: []string{"10.0.0.3"}},
					},
				},
			},
			expected: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
		},
		"should fall back to IP field when IPs is empty": {
			vmi: &kubeVirtV1.VirtualMachineInstance{
				Status: kubeVirtV1.VirtualMachineInstanceStatus{
					Interfaces: []kubeVirtV1.VirtualMachineInstanceNetworkInterface{
						{IP: "10.0.0.9"},
						{IP: "10.0.0.9"},
					},
				},
			},
			expected: []string{"10.0.0.9"},
		},
		"should return nil when all IP fields are empty": {
			vmi: &kubeVirtV1.VirtualMachineInstance{
				Status: kubeVirtV1.VirtualMachineInstanceStatus{
					Interfaces: []kubeVirtV1.VirtualMachineInstanceNetworkInterface{
						{IPs: []string{}},
						{IP: ""},
					},
				},
			},
			expected: nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tt.expected, extractIPAddresses(tt.vmi))
		})
	}
}

func TestExtractActivePods(t *testing.T) {
	tests := map[string]struct {
		vmi      *kubeVirtV1.VirtualMachineInstance
		expected []string
	}{
		"should return nil for nil vmi": {
			vmi:      nil,
			expected: nil,
		},
		"should return nil for empty active pods": {
			vmi:      &kubeVirtV1.VirtualMachineInstance{},
			expected: nil,
		},
		"should format pods with and without node name": {
			vmi: &kubeVirtV1.VirtualMachineInstance{
				Status: kubeVirtV1.VirtualMachineInstanceStatus{
					ActivePods: map[types.UID]string{
						types.UID("pod-a"): "node-1",
						types.UID("pod-b"): "",
						types.UID("pod-c"): "node-2",
					},
				},
			},
			expected: []string{"pod-a=node-1", "pod-b", "pod-c=node-2"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tt.expected, extractActivePods(tt.vmi))
		})
	}
}

func TestExtractBootOrder(t *testing.T) {
	first := uint(1)
	second := uint(2)
	tests := map[string]struct {
		disks    []kubeVirtV1.Disk
		expected []string
	}{
		"should return nil for empty disks": {
			disks:    nil,
			expected: nil,
		},
		"should ignore disks without boot order or name": {
			disks: []kubeVirtV1.Disk{
				{Name: "no-boot"},
				{BootOrder: &first},
			},
			expected: nil,
		},
		"should order by boot order then name": {
			disks: []kubeVirtV1.Disk{
				{Name: "disk-b", BootOrder: &first},
				{Name: "disk-a", BootOrder: &first},
				{Name: "disk-c", BootOrder: &second},
			},
			expected: []string{"disk-a=1", "disk-b=1", "disk-c=2"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tt.expected, extractBootOrder(tt.disks))
		})
	}
}

func TestExtractCDRomDisks(t *testing.T) {
	tests := map[string]struct {
		disks    []kubeVirtV1.Disk
		expected []string
	}{
		"should return nil for empty disks": {
			disks:    nil,
			expected: nil,
		},
		"should return only cdrom disks with names": {
			disks: []kubeVirtV1.Disk{
				{Name: "cd-1", DiskDevice: kubeVirtV1.DiskDevice{CDRom: &kubeVirtV1.CDRomTarget{}}},
				{Name: "disk-1", DiskDevice: kubeVirtV1.DiskDevice{Disk: &kubeVirtV1.DiskTarget{}}},
				{DiskDevice: kubeVirtV1.DiskDevice{CDRom: &kubeVirtV1.CDRomTarget{}}},
				{Name: "cd-1", DiskDevice: kubeVirtV1.DiskDevice{CDRom: &kubeVirtV1.CDRomTarget{}}},
			},
			expected: []string{"cd-1"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tt.expected, extractCDRomDisks(tt.disks))
		})
	}
}

func TestExtractDisksFromVM(t *testing.T) {
	boot := uint(1)
	disks := []kubeVirtV1.Disk{{Name: "disk-1", BootOrder: &boot}}
	tests := map[string]struct {
		vm       *kubeVirtV1.VirtualMachine
		expected []kubeVirtV1.Disk
	}{
		"should return nil for nil vm": {
			vm:       nil,
			expected: nil,
		},
		"should return nil for missing template": {
			vm:       &kubeVirtV1.VirtualMachine{},
			expected: nil,
		},
		"should return nil for empty disks": {
			vm: &kubeVirtV1.VirtualMachine{
				Spec: kubeVirtV1.VirtualMachineSpec{
					Template: &kubeVirtV1.VirtualMachineInstanceTemplateSpec{
						Spec: kubeVirtV1.VirtualMachineInstanceSpec{
							Domain: kubeVirtV1.DomainSpec{
								Devices: kubeVirtV1.Devices{
									Disks: nil,
								},
							},
						},
					},
				},
			},
			expected: nil,
		},
		"should return disks from template": {
			vm: &kubeVirtV1.VirtualMachine{
				Spec: kubeVirtV1.VirtualMachineSpec{
					Template: &kubeVirtV1.VirtualMachineInstanceTemplateSpec{
						Spec: kubeVirtV1.VirtualMachineInstanceSpec{
							Domain: kubeVirtV1.DomainSpec{
								Devices: kubeVirtV1.Devices{
									Disks: disks,
								},
							},
						},
					},
				},
			},
			expected: disks,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tt.expected, extractDisksFromVM(tt.vm))
		})
	}
}

func TestExtractDisksFromVMI(t *testing.T) {
	boot := uint(1)
	disks := []kubeVirtV1.Disk{{Name: "disk-1", BootOrder: &boot}}
	tests := map[string]struct {
		vmi      *kubeVirtV1.VirtualMachineInstance
		expected []kubeVirtV1.Disk
	}{
		"should return nil for nil vmi": {
			vmi:      nil,
			expected: nil,
		},
		"should return nil for empty disks": {
			vmi: &kubeVirtV1.VirtualMachineInstance{
				Spec: kubeVirtV1.VirtualMachineInstanceSpec{
					Domain: kubeVirtV1.DomainSpec{
						Devices: kubeVirtV1.Devices{
							Disks: nil,
						},
					},
				},
			},
			expected: nil,
		},
		"should return disks from vmi spec": {
			vmi: &kubeVirtV1.VirtualMachineInstance{
				Spec: kubeVirtV1.VirtualMachineInstanceSpec{
					Domain: kubeVirtV1.DomainSpec{
						Devices: kubeVirtV1.Devices{
							Disks: disks,
						},
					},
				},
			},
			expected: disks,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tt.expected, extractDisksFromVMI(tt.vmi))
		})
	}
}
