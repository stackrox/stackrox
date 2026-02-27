package common

import "github.com/stackrox/rox/generated/storage"

// VMScanParts groups the scan, components, and CVEs that together represent
// a single VM scan result. UpsertScan consumes this structure.
type VMScanParts struct {
	Scan       *storage.VirtualMachineScanV2
	Components []*storage.VirtualMachineComponentV2
	CVEs       []*storage.VirtualMachineCVEV2
}
