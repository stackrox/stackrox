package virtualmachine

import (
	"strings"

	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	pkgVM "github.com/stackrox/rox/pkg/virtualmachine"
)

const rhelGuestOS = "Red Hat Enterprise Linux"

// ResponseMeta.facts keys written by roxagent (proto enum String() values).
const (
	responseDetectedOSKey        = "detected_os"
	responseOSVersionKey         = "os_version"
	responseActivationStatusKey  = "activation_status"
	responseDNFMetadataStatusKey = "dnf_metadata_status"
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
	// GuestOSKey is informer-owned; roxagent reports DetectedGuestOSKey instead.
	for k, v := range vm.AgentFacts {
		if k == pkgVM.GuestOSKey {
			continue
		}
		facts[k] = v
	}
	return facts
}

// AgentFactsFromResponseFacts maps roxagent ResponseMeta.facts to user-facing
// fact keys. Unrecognized or unspecified enum strings are omitted so Sensor
// does not surface raw proto names in the UI.
func AgentFactsFromResponseFacts(facts map[string]string) map[string]string {
	if len(facts) == 0 {
		return nil
	}

	out := map[string]string{}
	switch facts[responseDetectedOSKey] {
	case v1.DetectedOS_RHEL.String():
		guestOS := rhelGuestOS
		if osVersion := facts[responseOSVersionKey]; osVersion != "" {
			guestOS += " " + osVersion
		}
		out[pkgVM.DetectedGuestOSKey] = guestOS
	}
	switch facts[responseActivationStatusKey] {
	case v1.ActivationStatus_ACTIVE.String():
		out[pkgVM.ActivationStatusKey] = pkgVM.ActivationStatusActive
	case v1.ActivationStatus_INACTIVE.String():
		out[pkgVM.ActivationStatusKey] = pkgVM.ActivationStatusInactive
	}
	switch facts[responseDNFMetadataStatusKey] {
	case v1.DnfMetadataStatus_AVAILABLE.String():
		out[pkgVM.DNFMetadataStatusKey] = pkgVM.DNFMetadataStatusAvailable
	case v1.DnfMetadataStatus_UNAVAILABLE.String():
		out[pkgVM.DNFMetadataStatusKey] = pkgVM.DNFMetadataStatusUnavailable
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// State is RUNNING when the VM instance is up, otherwise STOPPED.
func State(vm *Info) v1.VirtualMachine_State {
	if vm == nil {
		return v1.VirtualMachine_UNKNOWN
	}
	if vm.Running {
		return v1.VirtualMachine_RUNNING
	}
	return v1.VirtualMachine_STOPPED
}

// VSockCID returns the CID and whether it is set.
func VSockCID(vm *Info) (int32, bool) {
	if vm == nil || vm.VSOCKCID == nil {
		return 0, false
	}
	return int32(*vm.VSOCKCID), true
}

// SensorEvent builds the VirtualMachine resource event sent to Central.
func SensorEvent(action central.ResourceAction, clusterID string, vm *Info) *central.SensorEvent {
	if vm == nil {
		return nil
	}
	vsockCID, vsockCIDSet := VSockCID(vm)
	return &central.SensorEvent{
		Id:     string(vm.ID),
		Action: action,
		Resource: &central.SensorEvent_VirtualMachine{
			VirtualMachine: &v1.VirtualMachine{
				Id:          string(vm.ID),
				Namespace:   vm.Namespace,
				Name:        vm.Name,
				ClusterId:   clusterID,
				VsockCid:    vsockCID,
				VsockCidSet: vsockCIDSet,
				State:       State(vm),
				Facts:       Facts(vm),
			},
		},
	}
}
