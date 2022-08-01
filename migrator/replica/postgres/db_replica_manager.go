package postgres

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// CurrentReplica - active postgres replica name
	CurrentReplica = "central_active"

	// RestoreReplica - restore postgres replica name
	RestoreReplica = "central_restore"

	// BackupReplica - backup postgres replica name
	BackupReplica = "central_backup"

	// PreviousReplica - previous postgres replica used for rollback
	PreviousReplica = "central_previous"

	// TempReplica - temp postgres replica
	TempReplica = "temp"
)

var (
	knownReplicas = set.NewStringSet(CurrentReplica, RestoreReplica, BackupReplica, PreviousReplica)

	log = logging.CurrentModule().Logger()
)

// DBReplicaManager - scans and manage database replicas within central.
type DBReplicaManager interface {
	// Scan - Looks for database replicas
	Scan() error

	// GetReplicaToMigrate -- retrieves the replica that needs moved to the actived database.
	GetReplicaToMigrate() (string, string, error)

	// Persist -- moves the replica database to be the active database.
	Persist(replica string) error
}
