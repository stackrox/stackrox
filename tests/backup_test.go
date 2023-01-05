package tests

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/backup"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/tar"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
	"gopkg.in/yaml.v3"
)

// Grab the backup DB and open it, ensuring that there are values for deployments
func TestBackup(t *testing.T) {
	setupNginxLatestTagDeployment(t)
	defer teardownNginxLatestTagDeployment(t)

	waitForDeployment(t, nginxDeploymentName)

	for _, includeCerts := range []bool{false, true} {
		t.Run(fmt.Sprintf("includeCerts=%t", includeCerts), func(t *testing.T) {
			doTestBackup(t, includeCerts)
		})
	}
}

func doTestBackup(t *testing.T, includeCerts bool) {
	postgresEnabled := false
	postgresEnabledVar := os.Getenv("ROX_POSTGRES_DATASTORE")

	if postgresEnabledVar == "true" {
		postgresEnabled = true
	}

	tmpZipDir := t.TempDir()
	zipFilePath := filepath.Join(tmpZipDir, "backup.zip")
	out, err := os.Create(zipFilePath)
	require.NoError(t, err)

	client := centralgrpc.HTTPClientForCentral(t)
	endpoint := "/db/backup"
	if includeCerts {
		endpoint = "/api/extensions/backup"
	}
	resp, err := client.Get(endpoint)
	require.NoError(t, err)
	defer utils.IgnoreError(resp.Body.Close)
	_, err = io.Copy(out, resp.Body)
	require.NoError(t, err)

	defer utils.IgnoreError(out.Close)

	zipFile, err := zip.OpenReader(zipFilePath)
	require.NoError(t, err)
	defer utils.IgnoreError(zipFile.Close)

	if !postgresEnabled {
		checkZipForRocks(t, zipFile)
	} else {
		checkZipForPostgres(t, zipFile)
		checkZipForPassword(t, zipFile, includeCerts)
	}
	checkZipForCerts(t, zipFile, includeCerts)
	checkZipForVersion(t, zipFile)
}

func checkZipForVersion(t *testing.T, zipFile *zip.ReadCloser) {
	versionFileEntry := getFileWithName(zipFile, backup.MigrationVersion)
	require.NotNil(t, versionFileEntry)
	reader, err := versionFileEntry.Open()
	require.NoError(t, err)
	bytes, err := io.ReadAll(reader)
	require.NoError(t, err)
	version := &migrations.MigrationVersion{}
	err = yaml.Unmarshal(bytes, version)
	require.NoError(t, err)
	assert.Equal(t, version.MainVersion, version.MainVersion)
	assert.Equal(t, migrations.CurrentDBVersionSeqNum(), version.SeqNum)
}

func checkZipForCerts(t *testing.T, zipFile *zip.ReadCloser, includeCerts bool) {
	files := getFilesInDir(zipFile, "keys")
	if !includeCerts {
		require.Empty(t, files)
		return
	}
	require.NotEmpty(t, files)

	require.Equal(t, len(files), 3)
	for _, f := range files {
		info := f.FileInfo()
		require.NotZero(t, info.Size())
		require.Equal(t, filepath.Ext(f.Name), ".pem")
	}
}

func checkZipForRocks(t *testing.T, zipFile *zip.ReadCloser) {
	// Open the tar file holding the rocks DB backup.
	rocksFileEntry := getFileWithName(zipFile, "rocks.db")
	require.NotNil(t, rocksFileEntry)
	rocksFile, err := rocksFileEntry.Open()
	require.NoError(t, err)

	// Dump the untar'd rocks file to a scratch directory.
	tmpBackupDir := t.TempDir()

	err = tar.ToPath(tmpBackupDir, rocksFile)
	require.NoError(t, err)
	require.NoError(t, rocksFile.Close())

	// Generate the backup files in the directory.
	opts := gorocksdb.NewDefaultOptions()
	backupEngine, err := gorocksdb.OpenBackupEngine(opts, tmpBackupDir)
	require.NoError(t, err)

	// Restore the db to another temp directory
	tmpDBDir := t.TempDir()
	err = backupEngine.RestoreDBFromLatestBackup(tmpDBDir, tmpDBDir, gorocksdb.NewRestoreOptions())
	require.NoError(t, err)

	// Check for errors on cleanup.
	require.NoError(t, os.RemoveAll(tmpBackupDir))
	require.NoError(t, os.RemoveAll(tmpDBDir))
}

func checkZipForPostgres(t *testing.T, zipFile *zip.ReadCloser) {
	// Open the dump file holding the Postgres backup.
	postgresFileEntry := getFileWithName(zipFile, "postgres.dump")
	require.NotNil(t, postgresFileEntry)
	_, err := postgresFileEntry.Open()
	require.NoError(t, err)
}

func checkZipForPassword(t *testing.T, zipFile *zip.ReadCloser, includeCerts bool) {
	files := getFilesInDir(zipFile, backup.DatabaseBaseFolder)
	if !includeCerts {
		require.Empty(t, files)
		return
	}
	require.NotEmpty(t, files)

	require.Equal(t, len(files), 1)
	for _, f := range files {
		info := f.FileInfo()
		require.NotZero(t, info.Size())
		require.Equal(t, f.FileInfo().Name(), backup.DatabasePassword)
	}
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

func getFilesInDir(zipFile *zip.ReadCloser, dir string) []*zip.File {
	var files []*zip.File
	for _, f := range zipFile.File {
		if path.Dir(f.Name) == dir {
			files = append(files, f)
		}
	}
	return files
}
