package runner

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/registry"
)

// Run runs the migrator.
func Run(db *bolt.DB) error {
	version, err := getCurrentVersion(db)
	if err != nil {
		return err
	}
	if version == nil {
		return nil
	}
	dbSeqNum := version.GetSeqNum()
	currSeqNum := registry.CurrentSeqNum()
	if dbSeqNum > currSeqNum {
		return fmt.Errorf("DB sequence number %d is greater than the latest one we have (%d). This means "+
			"the migration binary is likely out of date", dbSeqNum, currSeqNum)
	}
	if dbSeqNum == currSeqNum {
		log.WriteToStderr("DB is up to date. Nothing to do here.")
		return nil
	}
	return runMigrations()
}

// TODO(viswa): Implement this.
func runMigrations() error {
	return nil
}
