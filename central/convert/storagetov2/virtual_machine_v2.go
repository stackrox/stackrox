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
	notes := make([]v2.VMNote, 0, len(vm.GetNotes()))
	for _, n := range vm.GetNotes() {
		notes = append(notes, convertVMNote(n))
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

func convertVMNote(note storage.VirtualMachineV2_Note) v2.VMNote {
	switch note {
	case storage.VirtualMachineV2_MISSING_SCAN_DATA:
		return v2.VMNote_VM_NOTE_MISSING_SCAN_DATA
	case storage.VirtualMachineV2_MISSING_SCANNER:
		return v2.VMNote_VM_NOTE_MISSING_SCANNER
	case storage.VirtualMachineV2_SCAN_FAILED:
		return v2.VMNote_VM_NOTE_SCAN_FAILED
	case storage.VirtualMachineV2_MISSING_SIGNATURE:
		return v2.VMNote_VM_NOTE_MISSING_SIGNATURE
	case storage.VirtualMachineV2_MISSING_SIGNATURE_VERIFICATION_DATA:
		return v2.VMNote_VM_NOTE_MISSING_SIGNATURE_VERIFICATION_DATA
	default:
		return v2.VMNote_VM_NOTE_MISSING_METADATA
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

// VirtualMachineCVEV2ToRow converts a storage CVE V2 to an API CVE row.
func VirtualMachineCVEV2ToRow(cve *storage.VirtualMachineCVEV2) *v2.VMCVERow {
	if cve == nil {
		return nil
	}
	return &v2.VMCVERow{
		Cve:             cve.GetCveBaseInfo().GetCve(),
		Severity:        v2.VulnerabilitySeverity(cve.GetSeverity()),
		IsFixable:       cve.GetIsFixable(),
		Cvss:            cve.GetPreferredCvss(),
		NvdCvss:         cve.GetNvdcvss(),
		EpssProbability: cve.GetEpssProbability(),
		PublishedOn:     cve.GetCveBaseInfo().GetPublishedOn(),
		Summary:         cve.GetCveBaseInfo().GetSummary(),
		Link:            cve.GetCveBaseInfo().GetLink(),
		Advisory:        advisoryToProto(cve.GetAdvisory()),
	}
}

// VirtualMachineComponentV2ToRow converts a storage component V2 to an API component row.
func VirtualMachineComponentV2ToRow(comp *storage.VirtualMachineComponentV2) *v2.VMComponentRow {
	if comp == nil {
		return nil
	}
	scanStatus := v2.ScanStatus_SCANNED
	for _, n := range comp.GetNotes() {
		if n == storage.VirtualMachineComponentV2_UNSCANNED {
			scanStatus = v2.ScanStatus_NOT_SCANNED
			break
		}
	}
	return &v2.VMComponentRow{
		Id:         comp.GetId(),
		Name:       comp.GetName(),
		Version:    comp.GetVersion(),
		Source:     v2.SourceType(comp.GetSource()),
		ScanStatus: scanStatus,
		CveCount:   comp.GetCveCount(),
	}
}

func advisoryToProto(adv *storage.Advisory) *v2.Advisory {
	if adv == nil {
		return nil
	}
	return &v2.Advisory{
		Name: adv.GetName(),
		Link: adv.GetLink(),
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

// ConvertScanNote converts a storage scan note to a v2 API scan note.
func ConvertScanNote(note storage.VirtualMachineScanV2_Note) v2.VMScanNote {
	switch note {
	case storage.VirtualMachineScanV2_OS_UNKNOWN:
		return v2.VMScanNote_VM_SCAN_NOTE_OS_UNKNOWN
	case storage.VirtualMachineScanV2_OS_UNSUPPORTED:
		return v2.VMScanNote_VM_SCAN_NOTE_OS_UNSUPPORTED
	default:
		return v2.VMScanNote_VM_SCAN_NOTE_UNSET
	}
}
