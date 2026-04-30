package virtualmachine

import (
	"maps"
	"strings"

	pkgVM "github.com/stackrox/rox/pkg/virtualmachine"
)

// Facts builds the VM facts map sent to Central.
func Facts(vm *Info) map[string]string {
	if vm == nil {
		return nil
	}

	facts := map[string]string{
		pkgVM.GuestOSKey: pkgVM.UnknownGuestOS,
	}
	if vm.GuestOS != "" {
		facts[pkgVM.GuestOSKey] = vm.GuestOS
	}
	if vm.Description != "" {
		facts[pkgVM.DescriptionKey] = vm.Description
	}
	if vm.NodeName != "" {
		facts[pkgVM.NodeNameKey] = vm.NodeName
	}
	if len(vm.IPAddresses) > 0 {
		facts[pkgVM.IPAddressesKey] = strings.Join(vm.IPAddresses, ", ")
	}
	if len(vm.ActivePods) > 0 {
		facts[pkgVM.ActivePodsKey] = strings.Join(vm.ActivePods, ", ")
	}
	if len(vm.BootOrder) > 0 {
		facts[pkgVM.BootOrderKey] = strings.Join(vm.BootOrder, ", ")
	}
	if len(vm.CDRomDisks) > 0 {
		facts[pkgVM.CDRomDisksKey] = strings.Join(vm.CDRomDisks, ", ")
	}
	if vm.AgentFacts != nil {
		maps.Copy(facts, vm.AgentFacts)
	}
	return facts
}
