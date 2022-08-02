package rocksdb

import (
	"regexp"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// Indexes
	bleveIndex = "scorch.bleve"
	index      = "index"

	// CurrentReplica - active rocksdb replica name
	CurrentReplica = migrations.Current

	// RestoreReplica - restore rocksdb replica name
	RestoreReplica = ".restore"

	// BackupReplica - backup rocksdb replica name
	BackupReplica = ".backup"

	// PreviousReplica - previous rocksdb replica used for rollback
	PreviousReplica = ".previous"

	// TempReplica - temp rocksdb replica
	TempReplica = "temp"
)

var (
	upgradeRegex  = regexp.MustCompile(`^\.db-*`)
	restoreRegex  = regexp.MustCompile(`^\.restore-*`)
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
