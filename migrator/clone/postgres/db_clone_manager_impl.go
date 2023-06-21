package postgres

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/clone/metadata"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// dbCloneManagerImpl - scans and manage database clones within central.
type dbCloneManagerImpl struct {
	cloneMap             map[string]*metadata.DBClone
	forceRollbackVersion string
	adminConfig          *postgres.Config
	sourceMap            map[string]string
	gc                   migGorm.Config
}

// New - returns a new ready-to-use store.
func New(forceVersion string, adminConfig *postgres.Config, sourceMap map[string]string) DBCloneManager {
	return &dbCloneManagerImpl{
		cloneMap:             make(map[string]*metadata.DBClone),
		forceRollbackVersion: forceVersion,
		adminConfig:          adminConfig,
		sourceMap:            sourceMap,
	}
}

// Scan - checks the persistent data of central and gather the clone information
// from disk.
func (d *dbCloneManagerImpl) Scan() error {
	ctx := sac.WithAllAccess(context.Background())

	// Get a short-lived connection for the purposes of checking the version of the clone.
	ver, err := migVer.ReadVersionPostgres(ctx, CurrentClone)
	if err != nil {
		return err
	}
	log.Infof("clone %s is of version %v", CurrentClone, ver)
	d.cloneMap[CurrentClone] = metadata.NewPostgres(ver, CurrentClone)

	currClone, currExists := d.cloneMap[CurrentClone]
	if !currExists || currClone.GetMigVersion() == nil {
		log.Info("Cannot find the current database or it has no version, so we need to let it create.")
	} else {
		// If the database version is newer than the software version make the user explicitly state they want to rollback.
		// TODO:  Do we still want to do this `forceRollbackVersion` check?
		if version.CompareVersions(currClone.GetVersion(), version.GetMainVersion()) > 0 {
			// Force rollback is not requested.
			if d.forceRollbackVersion != version.GetMainVersion() {
				return errors.New(metadata.ErrForceUpgradeDisabled)
			}
		}

		// current sequence number == database sequence number -- All good
		// current sequence number != database sequence number BUT database min >= current min -- ALL Good
		// version min < current database min -- DO NOT ROLLBACK
		if currClone.GetMinimumSeqNum() > migrations.MinimumSupportedDBVersionSeqNum() {
			return errors.Errorf(metadata.ErrSoftwareNotCompatibleWithDatabase, migrations.MinimumSupportedDBVersionSeqNum(), currClone.GetMinimumSeqNum())
		}
	}

	// Check restore version
	// TODO(ROX-16975): remove or hide behind a flag for ACS hosted only
	restoreExists, err := d.databaseExists(RestoreClone)
	if restoreExists && err == nil {
		restoreClone, restoreExists := d.cloneMap[RestoreClone]
		if restoreExists {
			// Restore from a newer version of central
			if restoreClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(restoreClone.GetVersion(), version.GetMainVersion()) > 0 {
				return errors.Errorf(metadata.ErrUnableToRestore, restoreClone.GetVersion(), version.GetMainVersion())
			}
		}
	}

	log.Info("Postgres Database clones:")
	for k, v := range d.cloneMap {
		log.Infof("%s -> %v", k, v.GetMigVersion())
	}

	return nil
}

// TODO(ROX-16975): remove or hide behind a flag for ACS hosted only
func (d *dbCloneManagerImpl) safeRemove(clone string) {
	log.Infof("safeRemove -> %s", clone)

	// Drop the database for the clone
	err := pgadmin.DropDB(d.sourceMap, d.adminConfig, clone)
	if err != nil {
		log.Errorf("Unable to drop clone - %q", clone)
	}

	delete(d.cloneMap, clone)
}

func (d *dbCloneManagerImpl) contains(clone string) bool {
	_, ok := d.cloneMap[clone]
	return ok
}

func (d *dbCloneManagerImpl) databaseExists(clone string) (bool, error) {
	return pgadmin.CheckIfDBExists(d.adminConfig, clone)
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the database clone name, flag informing if Rocks should be used as well and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate(rocksVersion *migrations.MigrationVersion, restoreFromRocks bool) (string, bool, error) {
	log.Info("GetCloneToMigrate")

	// If a restore clone exists, our focus is to try to restore that database.
	if _, ok := d.cloneMap[RestoreClone]; ok || restoreFromRocks {
		if restoreFromRocks {
			// We are restoring from Rocks, so we need to start with fresh Postgres each time
			err := pgadmin.DropDB(d.sourceMap, d.adminConfig, RestoreClone)
			if err != nil {
				log.Errorf("Unable to drop clone - %q", RestoreClone)
				return "", false, err
			}
			d.cloneMap[RestoreClone] = metadata.NewPostgres(rocksVersion, RestoreClone)
			return RestoreClone, true, nil
		}
		return RestoreClone, false, nil
	}

	currClone, currExists := d.cloneMap[CurrentClone]

	// If the current Postgres version is less than Rocks version then we need to migrate rocks to postgres
	// If the versions are the same, but rocks has a more recent update then we need to migrate rocks to postgres
	// Otherwise we roll with Postgres->Postgres.  We use central_temp as that will get cleaned up if the migration
	// of Rocks -> Postgres fails so we can start fresh.
	if d.versionExists(rocksVersion) {
		log.Infof("A previously used version of Rocks exists -- %v", rocksVersion)
		// TODO:  probably need to figure out last Rocks->Postgres migration and update those migrations to truncate
		// the tables in question so we don't have to restart the migration each time.
		if !currExists || !d.versionExists(currClone.GetMigVersion()) {
			d.cloneMap[CurrentClone] = metadata.NewPostgres(nil, CurrentClone)
			return CurrentClone, true, nil
		}
	}

	// Only need to make a copy if the migrations need to be performed
	if d.rollbackEnabled() && currClone.GetSeqNum() != migrations.CurrentDBVersionSeqNum() {
		// This is a rollback.  The minimum sequence number check was performed in the scan, so if we are here, we
		// can safely assume that passed and we can proceed rolling our version back with a compatible version of the
		// central database.
		if version.CompareVersions(currClone.GetVersion(), version.GetMainVersion()) > 0 || currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() {
			log.Infof("rollback to %q", currClone.GetDatabaseName())
			// Force rollback
			return CurrentClone, false, nil
		}
	}

	log.Info("Fell through all checks to return current.")
	return CurrentClone, false, nil
}

func (d *dbCloneManagerImpl) versionExists(dbVersion *migrations.MigrationVersion) bool {
	if dbVersion != nil &&
		dbVersion.SeqNum != 0 &&
		dbVersion.MainVersion != "0" {
		return true
	}

	return false
}

// Persist - replaces current clone with upgraded one.
func (d *dbCloneManagerImpl) Persist(cloneName string) error {
	if !d.contains(cloneName) && cloneName != CurrentClone {
		utils.CrashOnError(errors.New("Unexpected clone to persist"))
	}
	log.Infof("Persisting upgraded clone: %s", cloneName)

	switch cloneName {
	case RestoreClone:
		// For a restore, we should analyze it to get the stats because pg_dump does not
		// contain that information.
		err := pgadmin.AnalyzeDatabase(d.adminConfig, cloneName)
		if err != nil {
			log.Warnf("unable to force analyze restore database:  %v", err)
		}

		return d.doPersist(cloneName, BackupClone)
	case CurrentClone:
		// No need to persist
	default:
		utils.CrashOnError(errors.Errorf("commit with unknown clone: %s", cloneName))
	}
	return nil
}

func (d *dbCloneManagerImpl) doPersist(cloneName string, prev string) error {
	log.Infof("doPersist clone = %q, prev = %q", cloneName, prev)

	var moveCurrent string
	// Remove prev clone if exist.
	if prev != "" {
		moveCurrent = prev
		d.safeRemove(prev)
		d.cloneMap[prev] = d.cloneMap[CurrentClone]
	} else {
		return errors.Errorf("Invalid empty database clone name")
	}

	err := d.moveClones(moveCurrent, cloneName)
	if err != nil {
		log.Errorf("unable to move clones: %v", err)
		return err
	}

	// This is the case where we created a Temp because we have no previous.  Need to cleanup
	// once we have successfully moved the DBs around.
	if moveCurrent != prev {
		err = pgadmin.DropDB(d.sourceMap, d.adminConfig, moveCurrent)
		if err != nil {
			log.Errorf("Unable to remove the temp DB (%s): %v", moveCurrent, err)
			return err
		}
	}

	err = pgadmin.AnalyzeDatabase(d.adminConfig, CurrentClone)
	if err != nil {
		log.Warnf("unable to force analyze restore database: %v", err)
	}

	return nil
}

// TODO(ROX-16975) -- remove this.  At that point all work will be performend in a single database with the possible
// exception of restores for a ACS hosted Postgres.
func (d *dbCloneManagerImpl) moveClones(previousClone, updatedClone string) error {
	// Connect to different database for admin functions
	connectPool, err := pgadmin.GetAdminPool(d.adminConfig)
	if err != nil {
		return err
	}
	// Close the admin connection pool
	defer connectPool.Close()

	// Wrap in a transaction so either both renames work or none work
	ctx, cancel := context.WithTimeout(context.Background(), pgadmin.PostgresQueryTimeout)
	defer cancel()
	conn, err := connectPool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// Start a transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	// Move the current to the previous clone if it exists
	exists, err := d.databaseExists(CurrentClone)
	if err != nil {
		return err
	}
	if exists {
		err = d.renameClone(ctx, tx, CurrentClone, previousClone)
		if err != nil {
			return err
		}
	} else {
		log.Infof("current clone %q does not exist, must be start up", CurrentClone)
	}

	// Now flip the clone to be the primary DB
	err = d.renameClone(ctx, tx, updatedClone, CurrentClone)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Info("Commit")
		return err
	}

	return nil
}

// TODO(ROX-16975) -- remove this.  At that point all work will be performend in a single database with the possible
// exception of restores for a ACS hosted Postgres.
func (d *dbCloneManagerImpl) renameClone(ctx context.Context, tx *postgres.Tx, srcClone, destClone string) error {
	// Move the current to the previous clone
	err := pgadmin.TerminateConnection(d.adminConfig, srcClone)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", srcClone, destClone))
	if err != nil {
		log.Errorf("Unable to switch to clone %q DB: %v", destClone, err)
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (d *dbCloneManagerImpl) rollbackEnabled() bool {
	// If we are upgrading from earlier version without a migration version, we cannot do rollback.
	currClone, currExists := d.cloneMap[CurrentClone]
	if !currExists {
		log.Info("Current clone does not exist so rollback is disabled.")
		return false
	}

	return currClone.GetSeqNum() != 0
}

// GetCurrentVersion -- gets the version of the current clone
func (d *dbCloneManagerImpl) GetCurrentVersion() *migrations.MigrationVersion {
	ctx := sac.WithAllAccess(context.Background())
	ver, err := migVer.ReadVersionPostgres(ctx, CurrentClone)
	if err != nil {
		return nil
	}

	return ver
}
