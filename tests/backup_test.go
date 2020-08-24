package tests

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/tar"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
)

const scratchPath = "backuptest"

// Grab the backup DB and open it, ensuring that there are values for deployments
func TestBackup(t *testing.T) {
	setupNginxLatestTagDeployment(t)
	defer teardownNginxLatestTagDeployment(t)

	waitForDeployment(t, nginxDeploymentName)

	out, err := os.Create("backup.zip")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove("backup.zip")
	}()

	client := testutils.HTTPClientForCentral(t)
	resp, err := client.Get("/db/backup")
	require.NoError(t, err)
	defer utils.IgnoreError(resp.Body.Close)
	_, err = io.Copy(out, resp.Body)
	require.NoError(t, err)

	defer utils.IgnoreError(out.Close)

	zipFile, err := zip.OpenReader("backup.zip")
	require.NoError(t, err)
	defer utils.IgnoreError(zipFile.Close)

	checkZipForRocks(t, zipFile)
}

func checkZipForRocks(t *testing.T, zipFile *zip.ReadCloser) {
	// Open the tar file holding the rocks DB backup.
	rocksFileEntry := getFileWithName(zipFile, "rocks.db")
	require.NotNil(t, rocksFileEntry)
	rocksFile, err := rocksFileEntry.Open()
	require.NoError(t, err)

	// Dump the untar'd rocks file to a scratch directory.
	tmpBackupDir, err := ioutil.TempDir("", scratchPath)
	require.NoError(t, err)
	defer utils.IgnoreError(func() error { return os.RemoveAll(tmpBackupDir) })

	err = tar.ToPath(tmpBackupDir, rocksFile)
	require.NoError(t, err)
	require.NoError(t, rocksFile.Close())

	// Generate the backup files in the directory.
	opts := gorocksdb.NewDefaultOptions()
	backupEngine, err := gorocksdb.OpenBackupEngine(opts, tmpBackupDir)
	require.NoError(t, err)

	// Restore the db to another temp directory
	tmpDBDir, err := ioutil.TempDir("", scratchPath)
	require.NoError(t, err)
	defer utils.IgnoreError(func() error { return os.RemoveAll(tmpDBDir) })
	err = backupEngine.RestoreDBFromLatestBackup(tmpDBDir, tmpDBDir, gorocksdb.NewRestoreOptions())
	require.NoError(t, err)

	// Check for errors on cleanup.
	require.NoError(t, os.RemoveAll(tmpBackupDir))
	require.NoError(t, os.RemoveAll(tmpDBDir))
}

func getFileWithName(zipFile *zip.ReadCloser, name string) *zip.File {
	var ret *zip.File
	for _, f := range zipFile.File {
		if f.Name == name {
			ret = f
			break
		}
	}
	return ret
}
