package migrations

import (
	"github.com/stackrox/rox/pkg/migrations/internal"
)

const (
	// CurrentDatabase - current database
	CurrentDatabase = "central_active"
	// BackupDatabase - backup database
	BackupDatabase = "central_backup"
	// RestoreDatabase - restore database
	RestoreDatabase = "central_restore"
)

// DBMountPath is the directory path (within a container) where database storage device is mounted.
func DBMountPath() string {
	return internal.DBMountPath
}

// GetCurrentClone - returns the current clone
func GetCurrentClone() string {
	return CurrentDatabase
}

// GetBackupClone - returns the backup clone
func GetBackupClone() string {
	return BackupDatabase
}

// GetRestoreClone - returns the restore clone
func GetRestoreClone() string {
	return RestoreDatabase
}
