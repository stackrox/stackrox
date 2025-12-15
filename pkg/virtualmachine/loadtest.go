package virtualmachine

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	// TestNamespace is the namespace used for auto-generated VMs in test mode.
	// Used by both Sensor and Central when ROX_VM_TEST_MODE is enabled.
	TestNamespace = "vm-load-test"

	// testModeNamespaceUUID is the namespace UUID for generating deterministic VM IDs.
	// This is the DNS namespace UUID (RFC 4122) used as a base for our VM IDs.
	testModeNamespaceUUID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
)

// GenerateTestModeVMID creates a deterministic UUID v5 based on the vsock CID.
// This ensures the same CID always produces the same VM ID, which is critical for
// consistency between Sensor (which auto-generates VMs in the store) and Central
// (which receives index reports and creates VMs in the database).
//
// Both components must use this function to ensure VM IDs match.
func GenerateTestModeVMID(cid uint32) string {
	namespace := uuid.MustParse(testModeNamespaceUUID)
	return uuid.NewSHA1(namespace, []byte(fmt.Sprintf("vm-cid-%d", cid))).String()
}

// GenerateTestModeVMName returns the standard name for a test mode VM based on its CID.
func GenerateTestModeVMName(cid uint32) string {
	return fmt.Sprintf("vm-%d", cid)
}
