package storagetov2

import (
	"github.com/stackrox/rox/central/views/common"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

// VirtualMachineV2ToListItem converts a storage VM V2 to an API list item.
func VirtualMachineV2ToListItem(vm *storage.VirtualMachineV2) *v2.VMListItem {
	if vm == nil {
		return nil
	}
	return &v2.VMListItem{
		Id:          vm.GetId(),
		Name:        vm.GetName(),
		Namespace:   vm.GetNamespace(),
		ClusterId:   vm.GetClusterId(),
		ClusterName: vm.GetClusterName(),
		GuestOs:     vm.GetGuestOs(),
		State:       convertVirtualMachineV2State(vm.GetState()),
		LastUpdated: vm.GetLastUpdated(),
	}
}

// VirtualMachineV2ToDetail converts a storage VM V2 to a detailed API response.
func VirtualMachineV2ToDetail(vm *storage.VirtualMachineV2) *v2.VMDetail {
	if vm == nil {
		return nil
	}
	notes := make([]string, 0, len(vm.GetNotes()))
	for _, n := range vm.GetNotes() {
		notes = append(notes, n.String())
	}
	return &v2.VMDetail{
		Id:          vm.GetId(),
		Name:        vm.GetName(),
		Namespace:   vm.GetNamespace(),
		ClusterId:   vm.GetClusterId(),
		ClusterName: vm.GetClusterName(),
		GuestOs:     vm.GetGuestOs(),
		State:       convertVirtualMachineV2State(vm.GetState()),
		LastUpdated: vm.GetLastUpdated(),
		Facts:       vm.GetFacts(),
		VsockCid:    vm.GetVsockCid(),
		Notes:       notes,
	}
}

func convertVirtualMachineV2State(state storage.VirtualMachineV2_State) v2.VirtualMachineV2State {
	switch state {
	case storage.VirtualMachineV2_STOPPED:
		return v2.VirtualMachineV2State_VM_STATE_STOPPED
	case storage.VirtualMachineV2_RUNNING:
		return v2.VirtualMachineV2State_VM_STATE_RUNNING
	default:
		return v2.VirtualMachineV2State_VM_STATE_UNKNOWN
	}
}

// SeverityCountsToProto converts view severity counts to the API proto message.
func SeverityCountsToProto(counts common.ResourceCountByCVESeverity) *v2.VulnCountBySeverity {
	if counts == nil {
		return &v2.VulnCountBySeverity{}
	}
	return &v2.VulnCountBySeverity{
		Critical:  fixabilityToProto(counts.GetCriticalSeverityCount()),
		Important: fixabilityToProto(counts.GetImportantSeverityCount()),
		Moderate:  fixabilityToProto(counts.GetModerateSeverityCount()),
		Low:       fixabilityToProto(counts.GetLowSeverityCount()),
		Unknown:   fixabilityToProto(counts.GetUnknownSeverityCount()),
	}
}

func fixabilityToProto(f common.ResourceCountByFixability) *v2.VulnFixableCount {
	if f == nil {
		return &v2.VulnFixableCount{}
	}
	return &v2.VulnFixableCount{
		Total:   int32(f.GetTotal()),
		Fixable: int32(f.GetFixable()),
	}
}
