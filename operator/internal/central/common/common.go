package common

import "fmt"

const (
	// CentralPVCObsoleteAnnotation represents Central PVC has been obsoleted.
	// Only used for test.
	CentralPVCObsoleteAnnotation = "platform.stackrox.io/obsolete-central-pvc"

	// DefaultCentralDBBackupPVCName is the default name for Central DB backup PVC
	DefaultCentralDBBackupPVCName = "central-db-backup"
)

// Derive a backup PVC name from the main PVC name. Used only to have it in one
// place.
func GetBackupClaimName(claimName string) string {
	return fmt.Sprintf("%s-backup", claimName)
}
