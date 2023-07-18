package rocksdb

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/clone/metadata"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version"
)

// dbCloneManagerImpl - scans and manage database clones within central.
type dbCloneManagerImpl struct {
	basePath             string
	cloneMap             map[string]*metadata.DBClone
	forceRollbackVersion string
}

// New - returns a new ready-to-use store.
func New(basePath string, forceVersion string) DBCloneManager {
	return &dbCloneManagerImpl{basePath: basePath, cloneMap: make(map[string]*metadata.DBClone), forceRollbackVersion: forceVersion}
}

// Scan - checks the persistent data of central and gather the clone information
// from disk.
func (d *dbCloneManagerImpl) Scan() error {
	files, err := os.ReadDir(d.basePath)
	if err != nil {
		return err
	}

	// We use clones to collect all db clones (directory starting with db- or .restore-) matching upgrade or restore pattern.
	// We maintain clones with a known link in cloneMap. All unknown clones are to be removed.
	clonesToRemove := set.NewStringSet()
	for _, f := range files {
		switch name := f.Name(); {
		case knownClones.Contains(name):
			path := d.getPath(name)
			fileInfo, err := os.Lstat(path)
			if err != nil {
				return err
			}
			ver, err := migrations.Read(path)
			if err != nil {
				return err
			}
			if fileInfo.Mode()&os.ModeSymlink != 0 {
				linkTo, err := os.Readlink(path)
				if err != nil {
					return err
				}
				linkTo = filepath.Base(linkTo)
				log.Infof("Found clone %s -> %s", name, linkTo)

				// Add checks for dangling symbolic link. It should not happen by itself.
				exist, err := fileutils.Exists(d.getPath(linkTo))
				if err != nil {
					return err
				}
				if exist {
					d.cloneMap[name] = metadata.New(linkTo, ver)
				} else {
					return errors.Errorf("Found dangling symbolic link %s -> %s", name, linkTo)
				}
			} else {
				d.cloneMap[name] = metadata.New(name, ver)
			}
		case upgradeRegex.MatchString(name):
			clonesToRemove.Add(name)
		case restoreRegex.MatchString(name):
			clonesToRemove.Add(name)
		}
	}

	currClone, currExists := d.cloneMap[CurrentClone]
	if currExists && (currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(currClone.GetVersion(), version.GetMainVersion()) > 0) {
		// If there is no previous clone or force rollback is not requested, we cannot downgrade.
		prevClone, prevExists := d.cloneMap[PreviousClone]
		if !prevExists {
			if currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum() || version.GetVersionKind(currClone.GetVersion()) == version.ReleaseKind && version.GetVersionKind(version.GetMainVersion()) == version.ReleaseKind {
				return errors.New(metadata.ErrNoPrevious)
			}
			return errors.New(metadata.ErrNoPreviousInDevEnv)
		}
		// Force rollback is not requested.
		if d.forceRollbackVersion != version.GetMainVersion() {
			return errors.New(metadata.ErrForceUpgradeDisabled)
		}
		// If previous clone does not match
		if prevClone.GetVersion() != version.GetMainVersion() {
			return errors.Errorf(metadata.ErrPreviousMismatchWithVersions, prevClone.GetVersion(), version.GetMainVersion())
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

	log.Info("Database clones:")
	for k, v := range d.cloneMap {
		log.Infof("%s -> %v", k, v.GetMigVersion())
	}

	return nil
}

func (d *dbCloneManagerImpl) safeRemove(clone string) {
	path := d.getPath(clone)
	utils.Should(migrations.SafeRemoveDBWithSymbolicLink(path))
	delete(d.cloneMap, clone)
}

func (d *dbCloneManagerImpl) contains(clone string) bool {
	_, ok := d.cloneMap[clone]
	return ok
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the clone link, path to database and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate() (string, string, error) {
	if restoreRepl, ok := d.cloneMap[RestoreClone]; ok {
		log.Info("Database restore directory found. Migrating restored database files.")
		if err := os.RemoveAll(filepath.Join(migrations.DBMountPath(), bleveIndex)); err != nil && !os.IsNotExist(err) {
			log.Error(err)
		}
		if err := os.RemoveAll(filepath.Join(migrations.DBMountPath(), index)); err != nil && !os.IsNotExist(err) {
			log.Error(err)
		}
		return RestoreClone, d.getPath(restoreRepl.GetDirName()), nil
	}

	currClone, currExists := d.cloneMap[CurrentClone]
	// If our focus is Postgres, and there is no Rocks current, then we can ignore Rocks
	if !currExists {
		log.Warn("cannot find current clone for RocksDB")
		return "", "", nil
	}

	prevClone, prevExists := d.cloneMap[PreviousClone]
	if d.rollbackEnabled() && currClone.GetVersion() != version.GetMainVersion() {
		// If previous clone has the same version as current version, the previous upgrade was not completed.
		// Central could be in a loop of booting up the service. So we should continue to run with current.
		if prevExists && currClone.GetVersion() == prevClone.GetVersion() {
			return CurrentClone, d.getPath(d.cloneMap[CurrentClone].GetDirName()), nil
		}
		if prevExists && (version.CompareVersions(currClone.GetVersion(), version.GetMainVersion()) > 0 || currClone.GetSeqNum() > migrations.CurrentDBVersionSeqNum()) {
			// Force rollback
			return PreviousClone, d.getPath(d.cloneMap[PreviousClone].GetDirName()), nil
		}

		if currClone.GetSeqNum() < migrations.LastRocksDBVersionSeqNum() {
			d.safeRemove(PreviousClone)

			// If the current DB is in the last RocksDB version, then we would not upgrade it further.
			// We do not need to create a temp clone. The current won't be modified anyway.
			if d.hasSpaceForRollback() {
				tempDir := ".db-" + uuid.NewV4().String()
				log.Info("Database rollback enabled. Copying database files and migrate it to current version.")
				// Copy directory: not following link, do not overwrite
				cmd := exec.Command("cp", "-Rp", d.getPath(d.cloneMap[CurrentClone].GetDirName()), d.getPath(tempDir))
				if output, err := cmd.CombinedOutput(); err != nil {
					_ = os.RemoveAll(d.getPath(tempDir))
					return "", "", errors.Wrapf(err, "failed to copy current db %s", output)
				}
				ver, err := migrations.Read(d.getPath(tempDir))
				if err != nil {
					_ = os.RemoveAll(d.getPath(tempDir))
					return "", "", err
				}
				d.cloneMap[TempClone] = metadata.New(tempDir, ver)
				return TempClone, d.getPath(d.cloneMap[TempClone].GetDirName()), nil
			}
		}

		// If the space is not enough to make a clone, continue to upgrade with current.
		return CurrentClone, d.getPath(d.cloneMap[CurrentClone].GetDirName()), nil
	}

	// Rollback from previous version.
	if prevExists && prevClone.GetVersion() == version.GetMainVersion() {
		return PreviousClone, d.getPath(prevClone.GetDirName()), nil
	}

	return CurrentClone, d.getPath(d.cloneMap[CurrentClone].GetDirName()), nil
}

// Persist - replaces current clone with upgraded one.
func (d *dbCloneManagerImpl) Persist(cloneName string) error {
	if !d.contains(cloneName) {
		utils.CrashOnError(errors.New("Unexpected clone to persist"))
	}
	log.Infof("Persisting upgraded clone: %s", cloneName)

	switch cloneName {
	case RestoreClone:
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
	// Remove prev clone if exist.
	if prev != "" {
		d.safeRemove(prev)

		// prev -> current
		if err := fileutils.AtomicSymlink(d.cloneMap[CurrentClone].GetDirName(), d.getPath(prev)); err != nil {
			return err
		}
		d.cloneMap[prev] = d.cloneMap[CurrentClone]
	}

	// current -> clone
	if err := fileutils.AtomicSymlink(d.cloneMap[cloneName].GetDirName(), d.getPath(CurrentClone)); err != nil {
		return err
	}

	currClone := d.cloneMap[CurrentClone].GetDirName()
	d.cloneMap[CurrentClone] = d.cloneMap[cloneName]

	if prev == "" {
		d.safeRemove(currClone)
	}

	// Remove clone symbolic link only, if exists.
	_ = os.Remove(d.getPath(cloneName))
	return nil
}

func (d *dbCloneManagerImpl) getPath(cloneLink string) string {
	return filepath.Join(d.basePath, cloneLink)
}

func (d *dbCloneManagerImpl) rollbackEnabled() bool {
	// If we are upgrading from earlier version without a migration version, we cannot do rollback.
	currClone, currExists := d.cloneMap[CurrentClone]
	if !currExists {
		// If our focus is Postgres, just log the error and ignore Rocks as that likely means no PVC
		log.Warn("cannot find current clone for RocksDB")

		return false
	}
	return currClone.GetSeqNum() != 0
}

func (d *dbCloneManagerImpl) hasSpaceForRollback() bool {
	currClone, currExists := d.cloneMap[CurrentClone]
	if !currExists {
		// If our focus is Postgres, just log the error and ignore Rocks
		log.Warn("cannot find current clone for RocksDB")

		return false
	}
	availableBytes, err := fsutils.AvailableBytesIn(d.basePath)
	if err != nil {
		log.Warnf("Fail to get available bytes in %s", d.basePath)
		return false
	}
	requiredBytes, err := fileutils.DirectorySize(d.getPath(currClone.GetDirName()))
	if err != nil {
		log.Warnf("Fail to directory size %s", d.getPath(currClone.GetDirName()))
		return false
	}

	hasSpace := float64(availableBytes) > float64(requiredBytes)*(1.0+migrations.CapacityMarginFraction)
	log.Infof("Central has space to create backup for rollback: %v, required: %d, available: %d with %f margin", hasSpace, requiredBytes, availableBytes, migrations.CapacityMarginFraction)
	return hasSpace
}

// GetVersion - gets the version of the RocksDB clone
func (d *dbCloneManagerImpl) GetVersion(cloneName string) *migrations.MigrationVersion {
	clone, cloneExists := d.cloneMap[cloneName]
	if cloneExists {
		return clone.GetMigVersion()
	}
	return nil
}

// GetDirName - gets the directory name of the clone
func (d *dbCloneManagerImpl) GetDirName(cloneName string) string {
	return d.cloneMap[cloneName].GetDirName()
}

// CheckForRestore - checks to see if a restore clone exists
func (d *dbCloneManagerImpl) CheckForRestore() bool {
	if _, ok := d.cloneMap[RestoreClone]; ok {
		return true
	}
	return false
}
