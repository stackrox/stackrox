package migrations

import (
	"path/filepath"

	"github.com/stackrox/rox/pkg/migrations/internal"
)

const (
	// Current is the current database in use.
	Current = "current"
	// PreviousClone is the symbolic link pointing to the previous databases.
	PreviousClone = ".previous"

	// CurrentDatabase - current database
	CurrentDatabase = "central_active"
	// PreviousDatabase - previous database
	PreviousDatabase = "central_previous"
	// BackupDatabase - backup database
	BackupDatabase = "central_backup"
	// RestoreDatabase - restore database
	RestoreDatabase = "central_restore"
)

// DBMountPath is the directory path (within a container) where database storage device is mounted.
func DBMountPath() string {
	return internal.DBMountPath
}

// CurrentPath is the link (within a container) to current migration directory. This directory contains
// databases and other migration related contents.
func CurrentPath() string {
	return filepath.Join(internal.DBMountPath, Current)
}

// GetCurrentClone - returns the current clone
func GetCurrentClone() string {
	return CurrentDatabase
}

// GetBackupClone - returns the backup clone
func GetBackupClone() string {
	return BackupDatabase
}

// GetPreviousClone - returns the previous clone
func GetPreviousClone() string {
	return PreviousDatabase
}

// GetRestoreClone - returns the restore clone
func GetRestoreClone() string {
	return RestoreDatabase
}
