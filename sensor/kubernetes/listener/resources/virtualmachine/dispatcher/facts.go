package dispatcher

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

var descriptionAnnotationKeys = []string{
	"description",
	"openshift.io/description",
	"kubevirt.io/description",
}

func descriptionFromAnnotations(annotations map[string]string) string {
	if len(annotations) == 0 {
		return ""
	}
	values := make([]string, 0, len(descriptionAnnotationKeys))
	for _, key := range descriptionAnnotationKeys {
		if value := annotations[key]; value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return values[0]
	}
	return strings.Join(values, "; ")
}

func extractIPAddresses(vmi *kubeVirtV1.VirtualMachineInstance) []string {
	if vmi == nil || len(vmi.Status.Interfaces) == 0 {
		return nil
	}
	ips := set.NewStringSet()
	for _, iface := range vmi.Status.Interfaces {
		if len(iface.IPs) > 0 {
			ips.AddAll(iface.IPs...)
			continue
		}
		if iface.IP != "" {
			ips.Add(iface.IP)
		}
	}
	return ips.AsSortedSlice(func(i, j string) bool {
		return i < j
	})
}

func extractActivePods(vmi *kubeVirtV1.VirtualMachineInstance) []string {
	if vmi == nil || len(vmi.Status.ActivePods) == 0 {
		return nil
	}
	pods := set.NewStringSet()
	for podUID, nodeName := range vmi.Status.ActivePods {
		if nodeName == "" {
			pods.Add(string(podUID))
			continue
		}
		pods.Add(fmt.Sprintf("%s=%s", podUID, nodeName))
	}
	return pods.AsSortedSlice(func(i, j string) bool {
		return i < j
	})
}

func extractBootOrder(disks []kubeVirtV1.Disk) []string {
	type entry struct {
		name  string
		order uint
	}
	if len(disks) == 0 {
		return nil
	}
	entries := make([]entry, 0, len(disks))
	for _, disk := range disks {
		if disk.BootOrder == nil || disk.Name == "" {
			continue
		}
		entries = append(entries, entry{name: disk.Name, order: *disk.BootOrder})
	}
	if len(entries) == 0 {
		return nil
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if a.order == b.order {
			return cmp.Compare(a.name, b.name)
		}
		return cmp.Compare(a.order, b.order)
	})
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, fmt.Sprintf("%s=%d", entry.name, entry.order))
	}
	return out
}

func extractCDRomDisks(disks []kubeVirtV1.Disk) []string {
	if len(disks) == 0 {
		return nil
	}
	names := set.NewStringSet()
	for _, disk := range disks {
		if disk.CDRom != nil && disk.Name != "" {
			names.Add(disk.Name)
		}
	}
	return names.AsSortedSlice(func(i, j string) bool {
		return i < j
	})
}

func extractDisksFromVM(vm *kubeVirtV1.VirtualMachine) []kubeVirtV1.Disk {
	if vm == nil || vm.Spec.Template == nil {
		return nil
	}
	disks := vm.Spec.Template.Spec.Domain.Devices.Disks
	if len(disks) == 0 {
		return nil
	}
	return append([]kubeVirtV1.Disk(nil), disks...)
}

func extractDisksFromVMI(vmi *kubeVirtV1.VirtualMachineInstance) []kubeVirtV1.Disk {
	if vmi == nil {
		return nil
	}
	disks := vmi.Spec.Domain.Devices.Disks
	if len(disks) == 0 {
		return nil
	}
	return append([]kubeVirtV1.Disk(nil), disks...)
}
