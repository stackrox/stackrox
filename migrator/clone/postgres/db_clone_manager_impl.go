package postgres

import (
	"context"
	"fmt"
	"math"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/clone/metadata"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// dbCloneManagerImpl - scans and manage database clones within central.
type dbCloneManagerImpl struct {
	cloneMap             map[string]*metadata.DBClone
	forceRollbackVersion string
	adminConfig          *postgres.Config
	sourceMap            map[string]string
	supportPrevious      bool
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

func (d *dbCloneManagerImpl) getVersion() (*migrations.MigrationVersion, error) {
	ctx := sac.WithAllAccess(context.Background())

	gc := migGorm.GetConfig()
	db, err := gc.ConnectDatabaseWithRetries()
	if err != nil {
		return nil, err
	}
	defer migGorm.Close(db)

	return migVer.ReadVersionGormDB(ctx, db)
}

func (d *dbCloneManagerImpl) ensureVersionCompatible(ver *migrations.MigrationVersion) error {
	if d.versionExists(ver) {
		// current sequence number == database sequence number -- All good
		// current sequence number != database sequence number BUT database min >= current min -- ALL Good
		// version min < current database min -- DO NOT ROLLBACK
		if ver.MinimumSeqNum > migrations.MinimumSupportedDBVersionSeqNum() {
			return errors.Errorf(metadata.ErrSoftwareNotCompatibleWithDatabase, migrations.MinimumSupportedDBVersionSeqNum(), ver.MinimumSeqNum)
		}
	}

	return nil
}

// Scan - checks the persistent data of central and gather the clone information
// from disk.
func (d *dbCloneManagerImpl) Scan() error {
	// Beginning in 4.2 we are transitioning away from having multiple copies of the database.  Rollbacks to
	// 4.1 and later will all occur within the working database.  As we transition we need to check the
	// version to determine if it is valid to have other databases.  For instance if we are upgrading from 4.0 to 4.2
	// then we need to create a `central_previous`.  However, if we are upgrading from 4.1 or later to 4.2 or later,
	// then we do not.  Additionally, if we are upgrading from 4.1 and a `central_previous` still exists, we need to
	// remove it for consistency.

	// Get the version of the working database
	ctx := sac.WithAllAccess(context.Background())
	if pgconfig.IsExternalDatabase() {
		ver, err := d.getVersion()
		if err != nil {
			return err
		}

		return d.ensureVersionCompatible(ver)
	}

	// Get the version of the active DB.
	ver, err := migVer.ReadVersionPostgres(ctx, CurrentClone)
	if err != nil {
		return err
	}
	d.cloneMap[CurrentClone] = metadata.NewPostgres(ver, CurrentClone)
	log.Infof("db is of version %v", ver)

	if err := d.ensureVersionCompatible(ver); err != nil {
		return err
	}

	// Check to see if we are coming from pre-4.1 version where we may need to create and maintain `central_previous`
	if version.CompareVersions(ver.MainVersion, migrations.LastPostgresPreviousVersion) < 0 {
		d.supportPrevious = true
	}

	// We use clones to collect all db clones (directory starting with db- or .restore-) matching upgrade or restore pattern.
	// We maintain clones with a known link in cloneMap. All unknown clones are to be removed.
	clonesToRemove := set.NewStringSet()
	clonesToRemove.Add(TempClone)
	//  Check for `central_previous`
	prevExists, err := pgadmin.CheckIfDBExists(d.adminConfig, PreviousClone)
	if err != nil {
		return err
	}
	if prevExists && d.supportPrevious {
		// Get a short-lived connection for the purposes of checking the version of the clone.
		ver, err := migVer.ReadVersionPostgres(ctx, PreviousClone)
		if err != nil {
			return err
		}
		log.Infof("clone %s is of version %v", PreviousClone, ver)

		d.cloneMap[PreviousClone] = metadata.NewPostgres(ver, PreviousClone)
	}

	// Check restore version
	restoreInProgress, err := pgadmin.CheckIfDBExists(d.adminConfig, RestoreClone)
	if err != nil {
		return err
	}
	if restoreInProgress {
		// Get a short-lived connection for the purposes of checking the version of the clone.
		ver, err := migVer.ReadVersionPostgres(ctx, RestoreClone)
		if err != nil {
			return err
		}
		log.Infof("clone %s is of version %v", RestoreClone, ver)

		d.cloneMap[RestoreClone] = metadata.NewPostgres(ver, RestoreClone)
	}
	restoreClone, restoreExists := d.cloneMap[RestoreClone]
	if restoreExists {
		// Restore from a newer version of central
		if restoreClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(restoreClone.GetVersion(), version.GetMainVersion()) > 0 {
			return errors.Errorf(metadata.ErrUnableToRestore, restoreClone.GetVersion(), version.GetMainVersion())
		}
	}

	// Remove unknown clones that are not in use
	for _, r := range d.cloneMap {
		clonesToRemove.Remove(r.GetDirName())
	}

	// Now clones contains only unknown clones
	for r := range clonesToRemove {
		d.safeRemove(r)
	}

	log.Info("Postgres Database clones:")
	for k, v := range d.cloneMap {
		log.Infof("%s -> %v", k, v.GetMigVersion())
	}

	return nil
}

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

func (d *dbCloneManagerImpl) shouldMigrateFromRocks(rocksVersion *migrations.MigrationVersion, pgVersion *migrations.MigrationVersion) bool {
	// If the current Postgres version is less than Rocks version then we need to migrate rocks to postgres
	// If the versions are the same, but rocks has a more recent update then we need to migrate rocks to postgres
	// Otherwise we roll with Postgres->Postgres.  We use central_temp as that will get cleaned up if the migration
	// of Rocks -> Postgres fails so we can start fresh.
	if d.versionExists(rocksVersion) {
		log.Infof("A previously used version of Rocks exists -- %v", rocksVersion)
		// If we have not started nor completed migrating from RocksDB we need to return the flag that we
		// need RocksDB
		if !d.versionExists(pgVersion) || pgVersion.SeqNum < migrations.LastRocksDBToPostgresVersionSeqNum() {
			return true
		}
	}

	return false
}

func (d *dbCloneManagerImpl) checkForRocksToExternal(rocksVersion *migrations.MigrationVersion, restoreFromRocks bool) (string, bool, error) {
	// If the current Postgres version is less than Rocks version then we need to migrate rocks to postgres
	// If the versions are the same, but rocks has a more recent update then we need to migrate rocks to postgres
	// Otherwise we roll with Postgres->Postgres.  We use central_temp as that will get cleaned up if the migration
	// of Rocks -> Postgres fails so we can start fresh.
	ver, err := d.getVersion()
	if err != nil {
		return "", false, err
	}
	log.Infof("db is of version %v", ver)

	migrateFromRocks := restoreFromRocks || d.shouldMigrateFromRocks(rocksVersion, ver)

	return d.adminConfig.ConnConfig.Database, migrateFromRocks, nil
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the database clone name, flag informing if Rocks should be used as well and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate(rocksVersion *migrations.MigrationVersion, restoreFromRocks bool) (string, bool, error) {
	log.Info("GetCloneToMigrate")
	if pgconfig.IsExternalDatabase() {
		return d.checkForRocksToExternal(rocksVersion, restoreFromRocks)
	}

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
	if !currExists {
		d.cloneMap[CurrentClone] = metadata.NewPostgres(nil, CurrentClone)
	}

	if d.shouldMigrateFromRocks(rocksVersion, currClone.GetMigVersion()) {
		return CurrentClone, true, nil
	}

	// TODO(ROX-18005) -- Remove the use of central_temp and central_previous as all work will be done in
	// central_active.
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

		d.safeRemove(PreviousClone)
		// This is an upgrade, we are going to use `central_temp` until ROX-18005 so that a rollback to the previous
		// version works.  At the point of ROX-18005 all upgrades and rollbacks will use a single database.
		if d.hasSpaceForRollback() && d.supportPrevious {
			// Create a temp clone for processing of current
			// If such a clone already exists then we were previously in the middle of processing
			exists, err := d.databaseExists(TempClone)
			if err != nil {
				log.Errorf("Unable to create temp clone, will use current clone: %v", err)
				// If we had an issue checking whether "temp" exists we will proceed with the CurrentClone.
				// Essentially we treat this the same as if we could not create "temp" and proceed with
				// the migration.
				return CurrentClone, false, nil
			}
			if !exists {
				err := pgadmin.CreateDB(d.sourceMap, d.adminConfig, CurrentClone, TempClone)

				// If for some reason, we cannot create a temp clone we will need to continue to upgrade
				// with the current and thus no fallback.
				if err != nil {
					log.Errorf("Unable to create temp clone, will use current clone: %v", err)
					return CurrentClone, false, nil
				}
			}
			d.cloneMap[TempClone] = metadata.NewPostgres(d.cloneMap[CurrentClone].GetMigVersion(), TempClone)
			return TempClone, false, nil

		}

		// If the space is not enough to make a clone, continue to upgrade with current.
		return CurrentClone, false, nil
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
	case TempClone:
		return d.doPersist(cloneName, PreviousClone)
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
		moveCurrent = TempClone
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

// TODO(ROX-18005) -- remove this.  At that point all work will be performend in a single database with the possible
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

// TODO(ROX-18005) -- remove this.  At that point all work will be performend in a single database with the possible
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

// TODO(ROX-18005) -- remove this.  At that point all work will be performed in a single database with the possible
// exception of restores for a ACS hosted Postgres.
func (d *dbCloneManagerImpl) hasSpaceForRollback() bool {
	currReplica, currExists := d.cloneMap[CurrentClone]
	if !currExists {
		log.Warn("cannot find current replica for Postgres.  Indicates initial creation")
		return false
	}

	// When using managed services, Postgres space is not a concern at this time.
	if env.ManagedCentral.BooleanSetting() {
		return true
	}

	availableBytes, err := pgadmin.GetRemainingCapacity(d.adminConfig)
	if err != nil {
		log.Warnf("Fail to get available bytes in Postgres")
		return false
	}

	currentDBBytes, err := pgadmin.GetDatabaseSize(d.adminConfig, currReplica.GetDatabaseName())
	if err != nil {
		log.Warnf("Fail to get database size %s.  %v", currReplica.GetDatabaseName(), err)
		return false
	}

	requiredBytes := int64(math.Ceil(float64(currentDBBytes) * (1.0 + migrations.CapacityMarginFraction)))
	hasSpace := float64(availableBytes) > float64(requiredBytes)
	log.Infof("Central has space to create backup for rollback: %v, required: %d, available: %d with %f margin", hasSpace, requiredBytes, availableBytes, migrations.CapacityMarginFraction)

	return hasSpace
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
