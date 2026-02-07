package virtualmachine

import (
	"strings"

	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// Keep the facts keys camelCase to match the style used elsewhere in the UI.
const (
	// FactsGuestOSKey records the guest OS name or "unknown".
	FactsGuestOSKey = "guestOS"
	// FactsDescriptionKey records the VM/VMI description from annotations.
	FactsDescriptionKey = "description"
	// FactsIPAddressesKey records IPs joined by ", ".
	FactsIPAddressesKey = "ipAddresses"
	// FactsActivePodsKey records active pod references joined by ", ".
	FactsActivePodsKey = "activePods"
	// FactsNodeNameKey records the node name where the VM is running.
	FactsNodeNameKey = "nodeName"
	// FactsBootOrderKey records disk boot order entries joined by ", ".
	FactsBootOrderKey = "bootOrder"
	// FactsCDRomDisksKey records CD-ROM disk names joined by ", ".
	FactsCDRomDisksKey = "cdRomDisks"
	// FactsDetectedOSKey records the detected OS string from discovered data.
	FactsDetectedOSKey = "detectedOS"
	// FactsOSVersionKey records the detected OS version string from discovered data.
	FactsOSVersionKey = "osVersion"
	// FactsActivationStatusKey records the activation status from discovered data.
	FactsActivationStatusKey = "activationStatus"
	// FactsDNFMetadataStatusKey records the DNF metadata status from discovered data.
	FactsDNFMetadataStatusKey = "dnfMetadataStatus"
	// FactsUnknownGuestOS is used when the guest OS is not known yet.
	FactsUnknownGuestOS = "unknown"
)

// BuildFacts returns a map of VM facts, merged with discovered facts.
func BuildFacts(vm *Info, discoveredFacts map[string]string) map[string]string {
	facts := map[string]string{
		FactsGuestOSKey: FactsUnknownGuestOS,
	}
	if vm == nil {
		return facts
	}
	if vm.GuestOS != "" {
		facts[FactsGuestOSKey] = vm.GuestOS
	}
	if vm.Description != "" {
		facts[FactsDescriptionKey] = vm.Description
	}
	if vm.NodeName != "" {
		facts[FactsNodeNameKey] = vm.NodeName
	}
	if len(vm.IPAddresses) > 0 {
		facts[FactsIPAddressesKey] = strings.Join(vm.IPAddresses, ", ")
	}
	if len(vm.ActivePods) > 0 {
		facts[FactsActivePodsKey] = strings.Join(vm.ActivePods, ", ")
	}
	if len(vm.BootOrder) > 0 {
		facts[FactsBootOrderKey] = strings.Join(vm.BootOrder, ", ")
	}
	if len(vm.CDRomDisks) > 0 {
		facts[FactsCDRomDisksKey] = strings.Join(vm.CDRomDisks, ", ")
	}
	for k, v := range discoveredFacts {
		facts[k] = v
	}
	return facts
}

// StateFromInfo derives the VM state from Info.
func StateFromInfo(vm *Info) virtualMachineV1.VirtualMachine_State {
	if vm == nil {
		return virtualMachineV1.VirtualMachine_UNKNOWN
	}
	if vm.Running {
		return virtualMachineV1.VirtualMachine_RUNNING
	}
	return virtualMachineV1.VirtualMachine_STOPPED
}

// VSockCIDFromInfo returns the VSOCK CID and whether it is set.
func VSockCIDFromInfo(vm *Info) (int32, bool) {
	if vm == nil || vm.VSOCKCID == nil {
		return 0, false
	}
	return int32(*vm.VSOCKCID), true
}
