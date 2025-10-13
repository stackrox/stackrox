package rocksdb

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/clone/metadata"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
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

	log.Info("Database clones:")
	for k, v := range d.cloneMap {
		log.Infof("%s -> %v", k, v.GetMigVersion())
	}

	return nil
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the clone link, path to database and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate() (string, string, error) {
	_, currExists := d.cloneMap[CurrentClone]
	// If our focus is Postgres, and there is no Rocks current, then we can ignore Rocks
	if !currExists {
		log.Warn("cannot find current clone for RocksDB")
		return "", "", nil
	}

	return CurrentClone, d.getPath(d.cloneMap[CurrentClone].GetDirName()), nil
}

func (d *dbCloneManagerImpl) getPath(cloneLink string) string {
	return filepath.Join(d.basePath, cloneLink)
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
