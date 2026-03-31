package common

import "github.com/stackrox/rox/generated/storage"

// VMScanParts groups the scan, components, and CVEs that together represent
// a single VM scan result. UpsertScan consumes this structure.
type VMScanParts struct {
	Scan       *storage.VirtualMachineScanV2
	Components []*storage.VirtualMachineComponentV2
	CVEs       []*storage.VirtualMachineCVEV2
	// SourceComponents holds the original v1 scan components used for hash
	// computation. These are the pre-split representations that contain
	// embedded vulnerabilities, allowing direct hashing without intermediate
	// structs.
	SourceComponents []*storage.EmbeddedVirtualMachineScanComponent
}
