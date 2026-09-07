package vsockserver

import (
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// MappingProvider supplies the current repository-to-CPE mapping regardless
// of which updater (Sensor-backed or URL-backed) owns it in this process.
type MappingProvider interface {
	Ready() bool
	Hash() string
	UpdatePath() pb.RepoCPEMappingUpdatePath
	Bytes() ([]byte, error)
	// Path returns a file matching Bytes(). It may write that file if a
	// background persist has not landed yet.
	Path() (string, error)
}

// MappingUpdater accepts new mapping content pushed in over VSOCK. Only the
// Sensor-backed updater implements this; a URL-backed agent's Handler keeps
// its updater reference nil and rejects SyncRepoCPEMapping instead.
type MappingUpdater interface {
	Update(content []byte) (updated bool, err error)
}

// ScanBusyGate keeps a mapping Sync from replacing the file a scan is reading.
type ScanBusyGate interface {
	MarkScanBusy()
	MarkScanIdleAndApplyPending()
}
