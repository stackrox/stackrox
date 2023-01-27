package postgres

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/clone/metadata"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// dbCloneManagerImpl - scans and manage database clones within central.
type dbCloneManagerImpl struct {
	cloneMap             map[string]*metadata.DBClone
	forceRollbackVersion string
	adminConfig          *pgxpool.Config
	sourceMap            map[string]string
	gc                   migGorm.Config
}

// New - returns a new ready-to-use store.
func New(forceVersion string, adminConfig *pgxpool.Config, sourceMap map[string]string) DBCloneManager {
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
	clones := pgadmin.GetDatabaseClones(d.adminConfig)
	ctx := sac.WithAllAccess(context.Background())

	// We use clones to collect all db clones (directory starting with db- or .restore-) matching upgrade or restore pattern.
	// We maintain clones with a known link in cloneMap. All unknown clones are to be removed.
	clonesToRemove := set.NewStringSet()
	for _, clone := range clones {
		switch name := clone; {
		case knownClones.Contains(name):
			// Get a short-lived connection for the purposes of checking the version of the clone.
			ver, err := migVer.ReadVersionPostgres(ctx, name)
			if err != nil {
				return err
			}
			log.Infof("clone %s is of version %v", name, ver)

			d.cloneMap[name] = metadata.NewPostgres(ver, name)
			log.Debugf("Closing the pool from scan %q", name)
		case name == TempClone:
			clonesToRemove.Add(name)
		}
	}

	currClone, currExists := d.cloneMap[CurrentClone]
	if !currExists || currClone.GetMigVersion() == nil {
		log.Info("Cannot find the current database or it has no version, so we need to let it create and ignore other clones.")
		return nil
	}
	if currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(currClone.GetVersion(), version.GetMainVersion()) > 0 {
		// If there is no previous clone or force rollback is not requested, we cannot downgrade.
		prevClone, prevExists := d.cloneMap[PreviousClone]
		if !prevExists && currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() {
			if version.GetVersionKind(currClone.GetVersion()) == version.ReleaseKind && version.GetVersionKind(version.GetMainVersion()) == version.ReleaseKind {
				return errors.New(metadata.ErrNoPrevious)
			}
			return errors.New(metadata.ErrNoPreviousInDevEnv)
		}

		// Force rollback is not requested.
		if d.forceRollbackVersion != version.GetMainVersion() {
			return errors.New(metadata.ErrForceUpgradeDisabled)
		}

		// If previous clone does not match
		if prevExists && prevClone.GetVersion() != version.GetMainVersion() {
			return errors.Errorf(metadata.ErrPreviousMismatchWithVersions, prevClone.GetVersion(), version.GetMainVersion())
		}
	}

	// Check restore version
	restoreClone, restoreExists := d.cloneMap[RestoreClone]
	if restoreExists {
		// Restore from a newer version of central
		if restoreClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(restoreClone.GetVersion(), version.GetMainVersion()) > 0 {
			return errors.Errorf(metadata.ErrUnableToRestore, restoreClone.GetVersion(), version.GetMainVersion())
		}
	}

	// Remove unknown clones that is not in use
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

func (d *dbCloneManagerImpl) databaseExists(clone string) bool {
	return pgadmin.CheckIfDBExists(d.adminConfig, clone)
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the database clone name, flag informing if Rocks should be used as well and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate(rocksVersion *migrations.MigrationVersion, restoreFromRocks bool) (string, bool, error) {
	log.Info("GetCloneToMigrate")

	// If a restore clone exists, our focus is to try to restore that database.
	if _, ok := d.cloneMap[RestoreClone]; ok || restoreFromRocks {
		if restoreFromRocks {
			d.cloneMap[RestoreClone] = metadata.NewPostgres(rocksVersion, RestoreClone)
			return RestoreClone, true, nil
		}
		return RestoreClone, false, nil
	}

	currClone, currExists := d.cloneMap[CurrentClone]

	// If the current Postgres version is less than Rocks version then we need to migrate rocks to postgres
	// If the versions are the same, but rocks has a more recent update then we need to migrate rocks to postgres
	// Otherwise we roll with Postgres->Postgres
	if d.rocksExists(rocksVersion) {
		log.Infof("A previously used version of Rocks exists -- %v", rocksVersion)
		if !currExists || currClone.GetMigVersion() == nil {
			return CurrentClone, true, nil
		}
	}

	prevClone, prevExists := d.cloneMap[PreviousClone]
	// Only need to make a copy if the migrations need to be performed
	if d.rollbackEnabled() && currClone.GetSeqNum() != migrations.CurrentDBVersionSeqNum() {
		// If previous clone has the same version as current version, the previous upgrade was not completed.
		// Central could be in a loop of booting up the service. So we should continue to run with current.
		if prevExists && currClone.GetVersion() == prevClone.GetVersion() {
			return CurrentClone, false, nil
		}
		if version.CompareVersions(currClone.GetVersion(), version.GetMainVersion()) > 0 || currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() {
			// Force rollback
			return PreviousClone, false, nil
		}

		d.safeRemove(PreviousClone)
		if d.hasSpaceForRollback() {
			// Create a temp clone for processing of current
			// If such a clone already exists then we were previously in the middle of processing
			if !d.databaseExists(TempClone) {
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

	// Rollback from previous version.
	if prevExists && prevClone.GetVersion() == version.GetMainVersion() {
		return PreviousClone, false, nil
	}

	log.Info("Fell through all checks to return current.")
	return CurrentClone, false, nil
}

func (d *dbCloneManagerImpl) rocksExists(rocksVersion *migrations.MigrationVersion) bool {
	if rocksVersion != nil &&
		rocksVersion.SeqNum != 0 &&
		rocksVersion.MainVersion != "0" {
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
	case PreviousClone:
		return d.doPersist(cloneName, "")
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

	return nil
}

func (d *dbCloneManagerImpl) moveClones(previousClone, updatedClone string) error {
	// Connect to different database for admin functions
	connectPool := pgadmin.GetAdminPool(d.adminConfig)
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

	// Move the current to the previous clone
	err = d.renameClone(ctx, tx, CurrentClone, previousClone)
	if err != nil {
		return err
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

func (d *dbCloneManagerImpl) renameClone(ctx context.Context, tx pgx.Tx, srcClone, destClone string) error {
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
