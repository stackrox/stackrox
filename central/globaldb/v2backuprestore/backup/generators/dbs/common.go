package dbs

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/fsutils"
)

const (
	// marginOfSafety is how much more free space we want available then the current DB space used before we perform a
	// backup.
	marginOfSafety = 0.5
)

func findTmpPath(dbSize int64, tmpLocation string) (string, error) {
	requiredBytes := float64(dbSize) * (1.0 + marginOfSafety)

	// Check tmp for space to produce a backup.
	tmpDir, err := os.MkdirTemp("", tmpLocation)
	if err != nil {
		return "", err
	}
	tmpBytesAvailable, err := fsutils.AvailableBytesIn(tmpDir)
	if err != nil {
		return "", errors.Wrapf(err, "unable to calculates size of %s", tmpDir)
	}
	if float64(tmpBytesAvailable) > requiredBytes {
		return tmpDir, nil
	}

	// If there isn't enough space there, try using PVC to create it.
	pvcDir, err := os.MkdirTemp(globaldb.PVCPath, tmpLocation)
	if err != nil {
		return "", err
	}
	pvcBytesAvailable, err := fsutils.AvailableBytesIn(pvcDir)
	if err != nil {
		return "", errors.Wrapf(err, "unable to calculates size of %s", pvcDir)
	}
	if float64(pvcBytesAvailable) > requiredBytes {
		return pvcDir, nil
	}

	// If neither had enough space, return an error.
	return "", errors.Errorf("required %f bytes of space, found %f bytes in %s and %f bytes on PVC, cannot backup", requiredBytes, float64(tmpBytesAvailable), os.TempDir(), float64(pvcBytesAvailable))
}
