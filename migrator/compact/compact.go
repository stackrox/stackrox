package compact

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	sizeBuffer = 4 * 1024 * 1024
)

func availableBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func determineLargeEnoughDir(currSize uint64) (string, error) {
	desiredSpace := currSize + sizeBuffer

	mountAvailBytes, err := availableBytes(migrations.DBMountPath)
	if err != nil {
		return "", errors.Wrap(err, "error getting available bytes for DB mount path")
	}
	if mountAvailBytes > desiredSpace {
		return migrations.DBMountPath, nil
	}

	tmpAvailBytes, err := availableBytes("/tmp")
	if err != nil {
		return "", errors.Wrap(err, "error getting available bytes for /tmp")
	}
	if tmpAvailBytes > desiredSpace {
		name, err := ioutil.TempDir("", "")
		if err != nil {
			return "", errors.Wrap(err, "could not create temp directory")
		}
		return name, nil
	}
	return "", fmt.Errorf("not enough disk space: (needed: %d, /tmp: %d, %s: %d)", desiredSpace, tmpAvailBytes, migrations.DBMountPath, mountAvailBytes)
}

func checkIfCompactionIsNeeded() (*config.Config, error) {
	conf, exists, err := config.ReadConfig()
	if err != nil {
		return conf, err
	}

	if !exists {
		log.WriteToStderr("compaction defaults to false in the absence of a central-config configmap")
		return nil, nil
	}

	if !*conf.Maintenance.Compaction.Enabled {
		log.WriteToStderr("compaction is not triggered based on the central-config configmap")
		return nil, nil
	}
	return conf, nil
}

func transferFromScratchToDevice(dst, src string) error {
	log.WriteToStderr("writing scratch file %q to device file %q", src, dst)
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return errors.Wrapf(err, "error creating file at %q", dst)
	}
	srcFile, err := os.OpenFile(src, os.O_RDWR, 0600)
	if err != nil {
		return errors.Wrapf(err, "error opening file at %q", src)
	}
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return errors.Wrapf(err, "error copying %q to %q", src, dst)
	}
	if err := dstFile.Close(); err != nil {
		return errors.Wrapf(err, "error closing file %q", dst)
	}
	if err := srcFile.Close(); err != nil {
		return errors.Wrapf(err, "error closing file %q", src)
	}
	if err := os.Remove(src); err != nil {
		return errors.Wrapf(err, "error removing file %q", src)
	}
	return nil
}

func checkCompactionThreshold(config *config.Config, dbSize uint64, db *bolt.DB) bool {
	threshold := config.Maintenance.Compaction.FreeFractionThreshold
	if threshold == nil {
		log.WriteToStderr("no compaction threshold is set. Will compact on every startup with enabled:true")
		return true
	}
	// Instead of using oldDB.Stats().FreeAlloc which requires one write txn, just compute
	dbFreeAllocBytes := os.Getpagesize() * db.Stats().FreePageN
	freeFraction := float64(dbFreeAllocBytes) / float64(dbSize)
	if freeFraction > *threshold {
		log.WriteToStderr("Free fraction of %0.4f (%d/%d) is > %0.4f. Continuing with compaction", freeFraction, dbFreeAllocBytes, dbSize, *threshold)
		return true
	}
	log.WriteToStderr("Free fraction of %0.4f (%d/%d) is < %0.4f. Will not compact", freeFraction, dbFreeAllocBytes, dbSize, *threshold)
	return false
}

// Compact attempts to compact the DB
func Compact() error {
	config, err := checkIfCompactionIsNeeded()
	if err != nil {
		return err
	}
	if config == nil {
		return nil
	}

	log.WriteToStderr("starting DB compaction")
	oldDB, err := bolthelpers.Load()
	if err != nil {
		return err
	}
	if oldDB == nil {
		log.WriteToStderr("no existing DB exists. Stopping DB compaction")
		return nil
	}
	oldDB.MmapFlags = mmapFlags
	defer func() {
		// Close can be called multiple times so this is for security
		_ = oldDB.Close()
	}()

	fi, err := os.Stat(bolthelpers.Path())
	if err != nil {
		return err
	}
	originalBoltDBFileSize := uint64(fi.Size())

	// Check threshold for compaction
	if needsCompaction := checkCompactionThreshold(config, originalBoltDBFileSize, oldDB); !needsCompaction {
		return nil
	}

	// Check to see if the PVC can hold another BoltDB. If not, then write it to scratch
	// we prefer the PVC because then we can do an atomic rename because it is the same device
	compactionDirPath, err := determineLargeEnoughDir(originalBoltDBFileSize)
	if err != nil {
		return errors.Wrap(err, "error finding disk to write compacted DB. Try resizing the PV or freeing scratch space")
	}

	compactedBoltDBFilePath := filepath.Join(compactionDirPath, "compacted.db")
	// Remove old files if necessary
	_ = os.Remove(compactedBoltDBFilePath)

	compactedDB, err := bolt.Open(compactedBoltDBFilePath, 0600, nil)
	if err != nil {
		return errors.Wrap(err, "error opening compacted DB")
	}
	compactedDB.NoSync = true
	compactedDB.NoFreelistSync = true
	compactedDB.FreelistType = bolt.FreelistMapType

	if err := compact(compactedDB, oldDB, *config.Maintenance.Compaction.BucketFillFraction); err != nil {
		if err := compactedDB.Close(); err != nil {
			log.WriteToStderr("error closing compacted DB: %v", err)
		}
		if err := oldDB.Close(); err != nil {
			log.WriteToStderr("error closing old DB: %v", err)
		}
		if err := os.RemoveAll(compactedBoltDBFilePath); err != nil {
			log.WriteToStderr("error removing compacted DB: %v", err)
		}
		return errors.Wrap(err, "error executing compaction")
	}

	if err := compactedDB.Sync(); err != nil {
		return errors.Wrap(err, "error syncing compacted DB")
	}

	if err := compactedDB.Close(); err != nil {
		return errors.Wrap(err, "error closing compacted DB")
	}
	if err := oldDB.Close(); err != nil {
		return errors.Wrap(err, "error closing old DB")
	}

	if compactionDirPath != migrations.DBMountPath {
		// Now that we have compacted the DB, see if it will fit on the same Device so we can atomically rename it
		// If it does not then we may need manual intervention otherwise, we could cause data loss
		fi, err = os.Stat(compactedBoltDBFilePath)
		if err != nil {
			return errors.Wrapf(err, "error running stat on the compacted path")
		}

		availableOnMountPath, err := availableBytes(migrations.DBMountPath)
		if err != nil {
			return errors.Wrapf(err, "unable to get available bytes for %q", migrations.DBMountPath)
		}

		if uint64(fi.Size()) > availableOnMountPath {
			return fmt.Errorf("not enough space to move the compacted DB to the device. Needed space = %d bytes, but available = %d bytes", fi.Size(), availableOnMountPath)
		}
		// generate filepath on device and overwrite compactedBoltDBFilePath
		newCompactedBoltDBFilePath := filepath.Join(migrations.DBMountPath, "compacted.db")

		if err := transferFromScratchToDevice(newCompactedBoltDBFilePath, compactedBoltDBFilePath); err != nil {
			return errors.Wrap(err, "error transfering file from scratch")
		}
		compactedBoltDBFilePath = newCompactedBoltDBFilePath
	}

	if err := os.Rename(compactedBoltDBFilePath, bolthelpers.Path()); err != nil {
		return errors.Wrap(err, "error renaming db file")
	}
	log.WriteToStderr("successfully completed DB compaction")
	return nil
}
