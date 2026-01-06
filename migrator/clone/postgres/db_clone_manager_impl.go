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
		// minimum sequence number from the database > current software sequence number -- DO NOT ROLLBACK
		// This implies an unsupported rollback where structure and data may have changed
		if ver.MinimumSeqNum > migrations.CurrentDBVersionSeqNum() {
			return errors.Errorf(metadata.ErrSoftwareNotCompatibleWithDatabase, migrations.CurrentDBVersionSeqNum(), ver.MinimumSeqNum, migrations.MinimumSupportedDBVersion())
		}
	}

	return nil
}

// Scan - checks the persistent data of central and gather the clone information
// from disk.
func (d *dbCloneManagerImpl) Scan() error {
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

	// We use clones to collect all db clones (directory starting with db- or .restore-) matching upgrade or restore pattern.
	// We maintain clones with a known link in cloneMap. All unknown clones are to be removed.
	clonesToRemove := set.NewStringSet()

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
		// Restore from an unsupported old version (but skip check for seqNum 0 which represents a fresh/empty database)
		if restoreClone.GetSeqNum() > 0 && restoreClone.GetSeqNum() < migrations.MinimumSupportedDBVersionSeqNum() {
			return errors.Errorf("Restoring from version %q (sequence number %d) is not supported. The minimum supported version is %s (sequence number %d)",
				restoreClone.GetVersion(), restoreClone.GetSeqNum(), migrations.MinimumSupportedDBVersion(), migrations.MinimumSupportedDBVersionSeqNum())
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

// GetCloneToMigrate - finds a clone to migrate.
func (d *dbCloneManagerImpl) GetCloneToMigrate() (string, error) {
	log.Info("GetCloneToMigrate")
	if pgconfig.IsExternalDatabase() {
		return d.adminConfig.ConnConfig.Database, nil
	}

	// If a restore clone exists, our focus is to try to restore that database.
	if _, ok := d.cloneMap[RestoreClone]; ok {
		return RestoreClone, nil
	}

	currClone, currExists := d.cloneMap[CurrentClone]
	if !currExists {
		d.cloneMap[CurrentClone] = metadata.NewPostgres(nil, CurrentClone)
	}

	// Only need to make a copy if the migrations need to be performed
	if d.rollbackEnabled() && currClone.GetSeqNum() != migrations.CurrentDBVersionSeqNum() {
		// This is a rollback.  The minimum sequence number check was performed in the scan, so if we are here, we
		// can safely assume that passed and we can proceed rolling our version back with a compatible version of the
		// central database.
		if version.CompareVersions(currClone.GetVersion(), version.GetMainVersion()) > 0 || currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() {
			log.Infof("rollback to %q", currClone.GetDatabaseName())
			// Force rollback
			return CurrentClone, nil
		}

		// If the space is not enough to make a clone, continue to upgrade with current.
		return CurrentClone, nil
	}

	log.Info("Fell through all checks to return current.")
	return CurrentClone, nil
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

func (d *dbCloneManagerImpl) doPersist(cloneName string, backup string) error {
	log.Infof("doPersist clone = %q, backup = %q", cloneName, backup)
	if backup == "" {
		return errors.New("no backup clone provided")
	}

	// Remove backup clone if exist.
	moveCurrent := backup
	d.safeRemove(backup)
	d.cloneMap[backup] = d.cloneMap[CurrentClone]

	err := d.moveClones(moveCurrent, cloneName)
	if err != nil {
		log.Errorf("unable to move clones: %v", err)
		return err
	}

	err = pgadmin.AnalyzeDatabase(d.adminConfig, CurrentClone)
	if err != nil {
		log.Warnf("unable to force analyze restore database: %v", err)
	}

	return nil
}

// This moves a restore clone to current
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
	tx, ctx, err := conn.Begin(ctx)
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

// This is used for restores.
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
