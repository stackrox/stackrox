package replica

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version"
)

const (
	currentReplica = migrations.Current

	// Restore
	restoreReplica = ".restore"
	backupReplica  = ".backup"

	// Rollback
	previousReplica = ".previous"

	tempReplica = "temp"

	// Indexes
	bleveIndex = "scorch.bleve"
	index      = "index"

	errNoPrevious         = "Downgrade is not supported. No previous database for force rollback."
	errNoPreviousInDevEnv = `
Downgrade is not supported.
We compare dev builds by their release tags. For example, 3.0.58.x-58-g848e7365da is greater than
3.0.58.x-57-g848e7365da. However if the dev builds are on diverged branches, the sequence could be wrong.
These builds are not comparable.

To address this:
1. if you are testing migration, you can merge or rebase to make sure the builds are not diverged; or
2. if you simply want to switch the image, you can disable upgrade rollback and bypass this check by:
kubectl -n stackrox set env deploy/central ROX_DONT_COMPARE_DEV_BUILDS=true
`
	errForceUpgradeDisabled         = "Central force rollback is disabled. If you want to force rollback to the database before last upgrade, please enable force rollback to current version in central config. Note: all data updates since last upgrade will be lost."
	errPreviousMismatchWithVersions = "Database downgrade is not supported. We can only rollback to the central version before last upgrade. Last upgrade %s, current version %s"
)

var (
	upgradeRegex  = regexp.MustCompile(`^\.db-*`)
	restoreRegex  = regexp.MustCompile(`^\.restore-*`)
	knownReplicas = set.NewStringSet(currentReplica, restoreReplica, backupReplica, previousReplica)

	log = logging.CurrentModule().Logger()
)

type dbReplica struct {
	dirName string
	migVer  *migrations.MigrationVersion
}

func (d *dbReplica) getVersion() string {
	return d.migVer.MainVersion
}

func (d *dbReplica) getSeqNum() int {
	return d.migVer.SeqNum
}

// DBReplicaManager scans and manage database replicas within central.
type DBReplicaManager struct {
	basePath             string
	replicaMap           map[string]*dbReplica
	forceRollbackVersion string
}

// Scan checks the persistent data of central and gather the replica information
// from disk.
func Scan(basePath string, forceVersion string) (*DBReplicaManager, error) {
	manager := DBReplicaManager{basePath: basePath, replicaMap: make(map[string]*dbReplica), forceRollbackVersion: forceVersion}

	files, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	// We use replicas to collect all db replicas (directory starting with db- or .restore-) matching upgrade or restore pattern.
	// We maintain replicas with a known link in replicaMap. All unknown replicas are to be removed.
	replicasToRemove := set.NewStringSet()
	for _, f := range files {
		switch name := f.Name(); {
		case knownReplicas.Contains(name):
			path := manager.getPath(name)
			fileInfo, err := os.Lstat(path)
			if err != nil {
				return nil, err
			}
			ver, err := migrations.Read(path)
			if err != nil {
				return nil, err
			}
			if fileInfo.Mode()&os.ModeSymlink != 0 {
				linkTo, err := os.Readlink(path)
				if err != nil {
					return nil, err
				}
				linkTo = filepath.Base(linkTo)
				log.Infof("Found replica %s -> %s", name, linkTo)

				// Add checks for dangling symbolic link. It should not happen by itself.
				exist, err := fileutils.Exists(manager.getPath(linkTo))
				if err != nil {
					return nil, err
				}
				if exist {
					manager.replicaMap[name] = &dbReplica{dirName: linkTo, migVer: ver}
				} else {
					return nil, errors.Errorf("Found dangling symbolic link %s -> %s", name, linkTo)
				}
			} else {
				manager.replicaMap[name] = &dbReplica{dirName: name, migVer: ver}
			}
		case upgradeRegex.MatchString(name):
			replicasToRemove.Add(name)
		case restoreRegex.MatchString(name):
			replicasToRemove.Add(name)
		}
	}

	currReplica, currExists := manager.replicaMap[currentReplica]
	if !currExists {
		return nil, errors.Errorf("Cannot find database at %s", filepath.Join(basePath, currentReplica))
	}
	if currReplica.getSeqNum() > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(currReplica.getVersion(), version.GetMainVersion()) > 0 {
		// If there is no previous replica or force rollback is not requested, we cannot downgrade.
		prevReplica, prevExists := manager.replicaMap[previousReplica]
		if !prevExists {
			if currReplica.getSeqNum() > migrations.CurrentDBVersionSeqNum() || version.GetVersionKind(currReplica.getVersion()) == version.ReleaseKind && version.GetVersionKind(version.GetMainVersion()) == version.ReleaseKind {
				return nil, errors.New(errNoPrevious)
			}
			return nil, errors.New(errNoPreviousInDevEnv)
		}
		// Force rollback is not requested.
		if manager.forceRollbackVersion != version.GetMainVersion() {
			return nil, errors.New(errForceUpgradeDisabled)
		}
		// If previous replica does not match
		if prevReplica.getVersion() != version.GetMainVersion() {
			return nil, errors.Errorf(errPreviousMismatchWithVersions, prevReplica.getVersion(), version.GetMainVersion())
		}
	}

	// Remove unknown replicas that is not in use
	for _, r := range manager.replicaMap {
		replicasToRemove.Remove(r.dirName)
	}

	// Now replicas contains only unknown replicas
	for r := range replicasToRemove {
		manager.safeRemove(r)
	}

	log.Debug("Database replicas:")
	for k, v := range manager.replicaMap {
		log.Debugf("%s -> %v", k, v)
	}

	return &manager, nil
}

func (d *DBReplicaManager) safeRemove(replica string) {
	path := d.getPath(replica)
	utils.Should(migrations.SafeRemoveDBWithSymbolicLink(path))
	delete(d.replicaMap, replica)
}

func (d *DBReplicaManager) contains(replica string) bool {
	_, ok := d.replicaMap[replica]
	return ok
}

// GetReplicaToMigrate finds a replica to migrate.
// It returns the replica link, path to database and error if fails.
func (d *DBReplicaManager) GetReplicaToMigrate() (string, string, error) {
	if restoreRepl, ok := d.replicaMap[restoreReplica]; ok {
		log.Info("Database restore directory found. Migrating restored database files.")
		if err := os.RemoveAll(filepath.Join(migrations.DBMountPath(), bleveIndex)); err != nil && !os.IsNotExist(err) {
			log.Error(err)
		}
		if err := os.RemoveAll(filepath.Join(migrations.DBMountPath(), index)); err != nil && !os.IsNotExist(err) {
			log.Error(err)
		}
		return restoreReplica, d.getPath(restoreRepl.dirName), nil
	}

	currReplica := d.replicaMap[currentReplica]
	prevReplica, prevExists := d.replicaMap[previousReplica]
	if d.rollbackEnabled() && currReplica.getVersion() != version.GetMainVersion() {
		// If previous replica has the same version as current version, the previous upgrade was not completed.
		// Central could be in a loop of booting up the service. So we should continue to run with current.
		if prevExists && currReplica.getVersion() == prevReplica.getVersion() {
			return currentReplica, d.getPath(d.replicaMap[currentReplica].dirName), nil
		}
		if version.CompareVersions(currReplica.getVersion(), version.GetMainVersion()) > 0 || currReplica.getSeqNum() > migrations.CurrentDBVersionSeqNum() {
			// Force rollback
			return previousReplica, d.getPath(d.replicaMap[previousReplica].dirName), nil
		}

		d.safeRemove(previousReplica)
		if d.hasSpaceForRollback() {
			tempDir := ".db-" + uuid.NewV4().String()
			log.Info("Database rollback enabled. Copying database files and migrate it to current version.")
			// Copy directory: not following link, do not overwrite
			cmd := exec.Command("cp", "-Rp", d.getPath(d.replicaMap[currentReplica].dirName), d.getPath(tempDir))
			if output, err := cmd.CombinedOutput(); err != nil {
				_ = os.RemoveAll(d.getPath(tempDir))
				return "", "", errors.Wrapf(err, "failed to copy current db %s", output)
			}
			ver, err := migrations.Read(d.getPath(tempDir))
			if err != nil {
				_ = os.RemoveAll(d.getPath(tempDir))
				return "", "", err
			}
			d.replicaMap[tempReplica] = &dbReplica{dirName: tempDir, migVer: ver}
			return tempReplica, d.getPath(d.replicaMap[tempReplica].dirName), nil
		}

		// If the space is not enough to make a replica, continue to upgrade with current.
		return currentReplica, d.getPath(d.replicaMap[currentReplica].dirName), nil
	}

	// Rollback from previous version.
	if prevExists && prevReplica.getVersion() == version.GetMainVersion() {
		return previousReplica, d.getPath(prevReplica.dirName), nil
	}

	return currentReplica, d.getPath(d.replicaMap[currentReplica].dirName), nil
}

// Persist replaces current replica with upgraded one.
func (d *DBReplicaManager) Persist(replica string) error {
	if !d.contains(replica) {
		utils.CrashOnError(errors.New("Unexpected replica to persist"))
	}
	log.Infof("Persisting upgraded replica: %s", replica)

	switch replica {
	case restoreReplica:
		return d.doPersist(replica, backupReplica)
	case currentReplica:
		// No need to persist
	case tempReplica:
		return d.doPersist(replica, previousReplica)
	case previousReplica:
		return d.doPersist(replica, "")
	default:
		utils.CrashOnError(errors.Errorf("commit with unknown replica: %s", replica))
	}
	return nil
}

func (d *DBReplicaManager) doPersist(replica string, prev string) error {
	// Remove prev replica if exist.
	if prev != "" {
		d.safeRemove(prev)

		// prev -> current
		if err := fileutils.AtomicSymlink(d.replicaMap[currentReplica].dirName, d.getPath(prev)); err != nil {
			return err
		}
		d.replicaMap[prev] = d.replicaMap[currentReplica]
	}

	// current -> replica
	if err := fileutils.AtomicSymlink(d.replicaMap[replica].dirName, d.getPath(currentReplica)); err != nil {
		return err
	}

	currReplica := d.replicaMap[currentReplica].dirName
	d.replicaMap[currentReplica] = d.replicaMap[replica]

	if prev == "" {
		d.safeRemove(currReplica)
	}

	// Remove replica symbolic link only, if exists.
	_ = os.Remove(d.getPath(replica))
	return nil
}

func (d *DBReplicaManager) getPath(replicaLink string) string {
	return filepath.Join(d.basePath, replicaLink)
}

func (d *DBReplicaManager) rollbackEnabled() bool {
	// If we are upgrading from earlier version without a migration version, we cannot do rollback.
	currReplica, currExists := d.replicaMap[currentReplica]
	if !currExists {
		utils.Should(errors.New("cannot find current replica"))
		return false
	}
	return features.UpgradeRollback.Enabled() && currReplica.getSeqNum() != 0
}

func (d *DBReplicaManager) hasSpaceForRollback() bool {
	currReplica, currExists := d.replicaMap[currentReplica]
	if !currExists {
		utils.Should(errors.New("cannot find current replica"))
		return false
	}
	availableBytes, err := fsutils.AvailableBytesIn(d.basePath)
	if err != nil {
		log.Warnf("Fail to get available bytes in %s", d.basePath)
		return false
	}
	requiredBytes, err := fileutils.DirectorySize(d.getPath(currReplica.dirName))
	if err != nil {
		log.Warnf("Fail to directory size %s", d.getPath(currReplica.dirName))
		return false
	}

	hasSpace := float64(availableBytes) > float64(requiredBytes)*(1.0+migrations.CapacityMarginFraction)
	log.Infof("Central has space to create backup for rollback: %v, required: %d, available: %d with %f margin", hasSpace, requiredBytes, availableBytes, migrations.CapacityMarginFraction)
	return hasSpace
}
