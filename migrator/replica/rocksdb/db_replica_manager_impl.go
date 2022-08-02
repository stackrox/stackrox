package rocksdb

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/replica/metadata"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version"
)

// DBReplicaManagerImpl - scans and manage database replicas within central.
type DBReplicaManagerImpl struct {
	basePath             string
	ReplicaMap           map[string]*metadata.DBReplica
	forceRollbackVersion string
}

// New - returns a new ready-to-use store.
func New(basePath string, forceVersion string) *DBReplicaManagerImpl {
	return &DBReplicaManagerImpl{basePath: basePath, ReplicaMap: make(map[string]*metadata.DBReplica), forceRollbackVersion: forceVersion}
}

// Scan - checks the persistent data of central and gather the replica information
// from disk.
func (d *DBReplicaManagerImpl) Scan() error {
	files, err := os.ReadDir(d.basePath)
	if err != nil {
		return err
	}

	// We use replicas to collect all db replicas (directory starting with db- or .restore-) matching upgrade or restore pattern.
	// We maintain replicas with a known link in replicaMap. All unknown replicas are to be removed.
	replicasToRemove := set.NewStringSet()
	for _, f := range files {
		switch name := f.Name(); {
		case knownReplicas.Contains(name):
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
				log.Infof("Found replica %s -> %s", name, linkTo)

				// Add checks for dangling symbolic link. It should not happen by itself.
				exist, err := fileutils.Exists(d.getPath(linkTo))
				if err != nil {
					return err
				}
				if exist {
					d.ReplicaMap[name] = metadata.New(linkTo, ver)
				} else {
					return errors.Errorf("Found dangling symbolic link %s -> %s", name, linkTo)
				}
			} else {
				d.ReplicaMap[name] = metadata.New(name, ver)
			}
		case upgradeRegex.MatchString(name):
			replicasToRemove.Add(name)
		case restoreRegex.MatchString(name):
			replicasToRemove.Add(name)
		}
	}

	currReplica, currExists := d.ReplicaMap[CurrentReplica]
	if !currExists {
		return errors.Errorf("Cannot find database at %s", filepath.Join(d.basePath, CurrentReplica))
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
	path := d.getPath(replica)
	utils.Should(migrations.SafeRemoveDBWithSymbolicLink(path))
	delete(d.ReplicaMap, replica)
}

func (d *DBReplicaManagerImpl) contains(replica string) bool {
	_, ok := d.ReplicaMap[replica]
	return ok
}

// GetReplicaToMigrate - finds a replica to migrate.
// It returns the replica link, path to database and error if fails.
func (d *DBReplicaManagerImpl) GetReplicaToMigrate() (string, string, error) {
	if restoreRepl, ok := d.ReplicaMap[RestoreReplica]; ok {
		log.Info("Database restore directory found. Migrating restored database files.")
		if err := os.RemoveAll(filepath.Join(migrations.DBMountPath(), bleveIndex)); err != nil && !os.IsNotExist(err) {
			log.Error(err)
		}
		if err := os.RemoveAll(filepath.Join(migrations.DBMountPath(), index)); err != nil && !os.IsNotExist(err) {
			log.Error(err)
		}
		return RestoreReplica, d.getPath(restoreRepl.GetDirName()), nil
	}

	currReplica := d.ReplicaMap[CurrentReplica]
	prevReplica, prevExists := d.ReplicaMap[PreviousReplica]
	if d.rollbackEnabled() && currReplica.GetVersion() != version.GetMainVersion() {
		// If previous replica has the same version as current version, the previous upgrade was not completed.
		// Central could be in a loop of booting up the service. So we should continue to run with current.
		if prevExists && currReplica.GetVersion() == prevReplica.GetVersion() {
			return CurrentReplica, d.getPath(d.ReplicaMap[CurrentReplica].GetDirName()), nil
		}
		if version.CompareVersions(currReplica.GetVersion(), version.GetMainVersion()) > 0 || currReplica.GetSeqNum() > migrations.CurrentDBVersionSeqNum() {
			// Force rollback
			return PreviousReplica, d.getPath(d.ReplicaMap[PreviousReplica].GetDirName()), nil
		}

		d.safeRemove(PreviousReplica)
		if d.hasSpaceForRollback() {
			tempDir := ".db-" + uuid.NewV4().String()
			log.Info("Database rollback enabled. Copying database files and migrate it to current version.")
			// Copy directory: not following link, do not overwrite
			cmd := exec.Command("cp", "-Rp", d.getPath(d.ReplicaMap[CurrentReplica].GetDirName()), d.getPath(tempDir))
			if output, err := cmd.CombinedOutput(); err != nil {
				_ = os.RemoveAll(d.getPath(tempDir))
				return "", "", errors.Wrapf(err, "failed to copy current db %s", output)
			}
			ver, err := migrations.Read(d.getPath(tempDir))
			if err != nil {
				_ = os.RemoveAll(d.getPath(tempDir))
				return "", "", err
			}
			d.ReplicaMap[TempReplica] = metadata.New(tempDir, ver)
			return TempReplica, d.getPath(d.ReplicaMap[TempReplica].GetDirName()), nil
		}

		// If the space is not enough to make a replica, continue to upgrade with current.
		return CurrentReplica, d.getPath(d.ReplicaMap[CurrentReplica].GetDirName()), nil
	}

	// Rollback from previous version.
	if prevExists && prevReplica.GetVersion() == version.GetMainVersion() {
		return PreviousReplica, d.getPath(prevReplica.GetDirName()), nil
	}

	return CurrentReplica, d.getPath(d.ReplicaMap[CurrentReplica].GetDirName()), nil
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
	// Remove prev replica if exist.
	if prev != "" {
		d.safeRemove(prev)

		// prev -> current
		if err := fileutils.AtomicSymlink(d.ReplicaMap[CurrentReplica].GetDirName(), d.getPath(prev)); err != nil {
			return err
		}
		d.ReplicaMap[prev] = d.ReplicaMap[CurrentReplica]
	}

	// current -> replica
	if err := fileutils.AtomicSymlink(d.ReplicaMap[replicaName].GetDirName(), d.getPath(CurrentReplica)); err != nil {
		return err
	}

	currReplica := d.ReplicaMap[CurrentReplica].GetDirName()
	d.ReplicaMap[CurrentReplica] = d.ReplicaMap[replicaName]

	if prev == "" {
		d.safeRemove(currReplica)
	}

	// Remove replica symbolic link only, if exists.
	_ = os.Remove(d.getPath(replicaName))
	return nil
}

func (d *DBReplicaManagerImpl) getPath(replicaLink string) string {
	return filepath.Join(d.basePath, replicaLink)
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
	currReplica, currExists := d.ReplicaMap[CurrentReplica]
	if !currExists {
		utils.Should(errors.New("cannot find current replica"))
		return false
	}
	availableBytes, err := fsutils.AvailableBytesIn(d.basePath)
	if err != nil {
		log.Warnf("Fail to get available bytes in %s", d.basePath)
		return false
	}
	requiredBytes, err := fileutils.DirectorySize(d.getPath(currReplica.GetDirName()))
	if err != nil {
		log.Warnf("Fail to directory size %s", d.getPath(currReplica.GetDirName()))
		return false
	}

	hasSpace := float64(availableBytes) > float64(requiredBytes)*(1.0+migrations.CapacityMarginFraction)
	log.Infof("Central has space to create backup for rollback: %v, required: %d, available: %d with %f margin", hasSpace, requiredBytes, availableBytes, migrations.CapacityMarginFraction)
	return hasSpace
}
