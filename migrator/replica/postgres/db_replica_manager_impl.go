package postgres

import (
	"fmt"
	"runtime"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/replica/metadata"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// DBReplicaManagerImpl - scans and manage database replicas within central.
type DBReplicaManagerImpl struct {
	ReplicaMap           map[string]*metadata.DBReplica
	forceRollbackVersion string
	adminConfig          *pgxpool.Config
	sourceMap            map[string]string
}

// New - returns a new ready-to-use store.
func New(forceVersion string, adminConfig *pgxpool.Config, sourceMap map[string]string) *DBReplicaManagerImpl {
	return &DBReplicaManagerImpl{
		ReplicaMap:           make(map[string]*metadata.DBReplica),
		forceRollbackVersion: forceVersion,
		adminConfig:          adminConfig,
		sourceMap:            sourceMap,
	}
}

// Scan - checks the persistent data of central and gather the replica information
// from disk.
func (d *DBReplicaManagerImpl) Scan() error {
	replicas := pgadmin.GetDatabaseReplicas(d.adminConfig)

	// We use replicas to collect all db replicas (directory starting with db- or .restore-) matching upgrade or restore pattern.
	// We maintain replicas with a known link in replicaMap. All unknown replicas are to be removed.
	replicasToRemove := set.NewStringSet()
	for _, replica := range replicas {
		switch name := replica; {
		case knownReplicas.Contains(name):
			// Get a short-lived connection for the purposes of checking the version of the replica.
			pool := pgadmin.GetReplicaPool(d.adminConfig, name)

			ver, err := migrations.ReadVersion(pool)
			if err != nil {
				return err
			}
			log.Debugf("replica %s is of version %v", name, ver)

			d.ReplicaMap[name] = metadata.NewPostgres(ver, name)
			log.Debugf("Closing the pool from scan %q", name)
			pool.Close()
		}
	}

	currReplica, currExists := d.ReplicaMap[CurrentReplica]
	if !currExists || currReplica.GetMigVersion() == nil {
		log.Info("Cannot find the current database or it has no version, so we need to let it create and ignore other replicas.")
		return nil
	}
	if currReplica.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(currReplica.GetVersion(), version.GetMainVersion()) > 0 {
		// If there is no previous replica or force rollback is not requested, we cannot downgrade.
		prevReplica, prevExists := d.ReplicaMap[PreviousReplica]
		if !prevExists {
			if currReplica.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.GetVersionKind(currReplica.GetVersion()) == version.ReleaseKind && version.GetVersionKind(version.GetMainVersion()) == version.ReleaseKind {
				return errors.New(metadata.ErrNoPrevious)
			}
			return errors.New(metadata.ErrNoPreviousInDevEnv)
		}

		// Force rollback is not requested.
		if d.forceRollbackVersion != version.GetMainVersion() {
			return errors.New(metadata.ErrForceUpgradeDisabled)
		}

		// If previous replica does not match
		if prevReplica.GetVersion() != version.GetMainVersion() {
			return errors.Errorf(metadata.ErrPreviousMismatchWithVersions, prevReplica.GetVersion(), version.GetMainVersion())
		}
	}

	// Check restore version
	restoreReplica, restoreExists := d.ReplicaMap[RestoreReplica]
	if restoreExists {
		// Restore from a newer version of central
		if restoreReplica.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(restoreReplica.GetVersion(), version.GetMainVersion()) > 0 {
			return errors.Errorf(metadata.ErrUnableToRestore, restoreReplica.GetVersion(), version.GetMainVersion())
		}
	}

	// Remove unknown replicas that is not in use
	for _, r := range d.ReplicaMap {
		replicasToRemove.Remove(r.GetDirName())
	}

	// Now replicas contains only unknown replicas
	for r := range replicasToRemove {
		d.safeRemove(r)
	}

	log.Debug("Database replicas:")
	for k, v := range d.ReplicaMap {
		log.Debugf("%s -> %v", k, v)
	}

	return nil
}

func (d *DBReplicaManagerImpl) safeRemove(replica string) {
	log.Infof("safeRemove -> %s", replica)
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		fmt.Printf("called from %s\n", details.Name())
	}

	// Drop the database for the replica
	err := pgadmin.DropDB(d.sourceMap, d.adminConfig, replica)
	if err != nil {
		log.Errorf("Unable to drop replica - %q", replica)
	}

	delete(d.ReplicaMap, replica)
}

func (d *DBReplicaManagerImpl) contains(replica string) bool {
	_, ok := d.ReplicaMap[replica]
	return ok
}

func (d *DBReplicaManagerImpl) databaseExists(replica string) bool {
	return pgadmin.CheckIfDBExists(d.adminConfig, replica)
}

// GetReplicaToMigrate - finds a replica to migrate.
// It returns the replica link, path to database and error if fails.
func (d *DBReplicaManagerImpl) GetReplicaToMigrate() (string, string, error) {
	log.Info("GetReplicatToMigrate")
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		fmt.Printf("called from %s\n", details.Name())
	}

	// If a restore replica exists, our focus is to try to restore that database.
	if _, ok := d.ReplicaMap[RestoreReplica]; ok {
		return RestoreReplica, "", nil
	}

	currReplica := d.ReplicaMap[CurrentReplica]
	prevReplica, prevExists := d.ReplicaMap[PreviousReplica]
	if d.rollbackEnabled() && currReplica.GetVersion() != version.GetMainVersion() {
		// If previous replica has the same version as current version, the previous upgrade was not completed.
		// Central could be in a loop of booting up the service. So we should continue to run with current.
		if prevExists && currReplica.GetVersion() == prevReplica.GetVersion() {
			return CurrentReplica, "", nil
		}
		if version.CompareVersions(currReplica.GetVersion(), version.GetMainVersion()) > 0 || currReplica.GetSeqNum() > migrations.CurrentDBVersionSeqNum() {
			// Force rollback
			return PreviousReplica, "", nil
		}

		d.safeRemove(PreviousReplica)
		if d.hasSpaceForRollback() {
			// Create a temp replica for processing of current
			// If such a replica already exists then we were previously in the middle of processing
			if !d.databaseExists(TempReplica) {
				err := pgadmin.CreateDB(d.sourceMap, d.adminConfig, CurrentReplica, TempReplica)

				// If for some reason, we cannot create a temp replica we will need to continue to upgrade
				// with the current and thus no fallback.
				if err != nil {
					log.Errorf("Unable to create Temp database: %v", err)
				}
			}
			d.ReplicaMap[TempReplica] = metadata.NewPostgres(d.ReplicaMap[CurrentReplica].GetMigVersion(), TempReplica)
			return TempReplica, "", nil

		}

		// If the space is not enough to make a replica, continue to upgrade with current.
		return CurrentReplica, "", nil
	}

	// Rollback from previous version.
	if prevExists && prevReplica.GetVersion() == version.GetMainVersion() {
		return PreviousReplica, "", nil
	}

	return CurrentReplica, "", nil
}

// Persist - replaces current replica with upgraded one.
func (d *DBReplicaManagerImpl) Persist(replicaName string) error {
	if !d.contains(replicaName) {
		utils.CrashOnError(errors.New("Unexpected replica to persist"))
	}
	log.Infof("Persisting upgraded replica: %s", replicaName)

	switch replicaName {
	case RestoreReplica:
		return d.doPersist(replicaName, BackupReplica)
	case CurrentReplica:
		// No need to persist
	case TempReplica:
		return d.doPersist(replicaName, PreviousReplica)
	case PreviousReplica:
		return d.doPersist(replicaName, "")
	default:
		utils.CrashOnError(errors.Errorf("commit with unknown replica: %s", replicaName))
	}
	return nil
}

func (d *DBReplicaManagerImpl) doPersist(replicaName string, prev string) error {
	log.Infof("doPersist replica = %q, prev = %q", replicaName, prev)

	// Connect to different database for admin functions
	connectPool := pgadmin.GetAdminPool(d.adminConfig)
	// Close the admin connection pool
	defer connectPool.Close()

	var moveCurrent string
	// Remove prev replica if exist.
	if prev != "" {
		moveCurrent = prev
		d.safeRemove(prev)
		d.ReplicaMap[prev] = d.ReplicaMap[CurrentReplica]
	} else {
		moveCurrent = "temp"
	}

	// Flip current to prev
	err := pgadmin.RenameDB(connectPool, CurrentReplica, moveCurrent)
	if err != nil {
		log.Errorf("Unable to switch to previous DB: %v", err)
		return err
	}

	// Now flip the replica to be the primary DB
	err = pgadmin.RenameDB(connectPool, replicaName, CurrentReplica)
	if err != nil {
		log.Errorf("Unable to switch to replica DB: %v", err)
		return err
	}

	// This is the case where we created a Temp because we have no previous.  Need to cleanup
	// once we have sccessfully moved the DBs around.
	if moveCurrent != prev {
		err = pgadmin.DropDB(d.sourceMap, d.adminConfig, moveCurrent)
		if err != nil {
			log.Errorf("Unable to remove the temp DB (%s): %v", moveCurrent, err)
			return err
		}
	}

	return nil
}

func (d *DBReplicaManagerImpl) rollbackEnabled() bool {
	// If we are upgrading from earlier version without a migration version, we cannot do rollback.
	currReplica, currExists := d.ReplicaMap[CurrentReplica]
	if !currExists {
		utils.Should(errors.New("cannot find current replica"))
		return false
	}
	return features.UpgradeRollback.Enabled() && currReplica.GetSeqNum() != 0
}

func (d *DBReplicaManagerImpl) hasSpaceForRollback() bool {
	// TODO:  Figure out what this means in the Postgres world.  Probably a separate ticket.
	return true
}
