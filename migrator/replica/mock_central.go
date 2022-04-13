package replica

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/fileutils"
	"github.com/stackrox/stackrox/pkg/migrations"
	migrationtestutils "github.com/stackrox/stackrox/pkg/migrations/testutils"
	"github.com/stackrox/stackrox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	breakAfterScan           = "scan"
	breakAfterGetReplica     = "get-replica"
	breakBeforePersist       = "persist"
	breakAfterRemove         = "remove"
	breakBeforeCommitCurrent = "current"
	breakBeforeCleanUp       = "cleanup"
)

type versionPair struct {
	version string
	seqNum  int
}

type mockCentral struct {
	t               *testing.T
	mountPath       string
	rollbackEnabled bool
	// Set version function has to be provided by test itself.
	setVersion func(t *testing.T, ver *versionPair)
}

func createCentral(t *testing.T) *mockCentral {
	mountDir, err := os.MkdirTemp("", "mock-central-")
	require.NoError(t, err)
	mock := mockCentral{t: t, mountPath: mountDir}
	dbPath := filepath.Join(mountDir, ".db-init")
	require.NoError(t, os.Mkdir(dbPath, 0755))
	require.NoError(t, os.Symlink(dbPath, filepath.Join(mountDir, "current")))
	migrationtestutils.SetDBMountPath(t, mountDir)
	return &mock
}

func (m *mockCentral) destroyCentral() {
	_ = os.RemoveAll(m.mountPath)
}

func (m *mockCentral) enableRollBack(enable bool) {
	m.rollbackEnabled = enable
	require.NoError(m.t, os.Setenv(features.UpgradeRollback.EnvVar(), strconv.FormatBool(enable)))
}

func (m *mockCentral) rebootCentral() {
	curSeq := migrations.CurrentDBVersionSeqNum()
	curVer := version.GetMainVersion()
	m.runMigrator("", "")
	m.runCentral()
	assert.Equal(m.t, curSeq, migrations.CurrentDBVersionSeqNum())
	assert.Equal(m.t, curVer, version.GetMainVersion())
}

func (m *mockCentral) migrateWithVersion(ver *versionPair, breakpoint string, forceRollback string) {
	m.setVersion(m.t, ver)
	m.runMigrator(breakpoint, forceRollback)
}

func (m *mockCentral) upgradeCentral(ver *versionPair, breakpoint string) {
	curVer := &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()}
	m.migrateWithVersion(ver, breakpoint, "")
	// Re-run migrator if the previous one breaks
	if breakpoint != "" {
		m.runMigrator("", "")
	}
	m.runCentral()
	if m.rollbackEnabled && version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
		m.verifyReplica(previousReplica, curVer)
	} else {
		assert.NoDirExists(m.t, filepath.Join(m.mountPath, previousReplica))
	}
}

func (m *mockCentral) upgradeDB(path string) {
	// Verify no downgrade
	if exists, _ := fileutils.Exists(filepath.Join(path, "db")); exists {
		data, err := os.ReadFile(filepath.Join(path, "db"))
		require.NoError(m.t, err)
		currDBSeq, err := strconv.Atoi(string(data))
		require.NoError(m.t, err)
		require.LessOrEqual(m.t, currDBSeq, migrations.CurrentDBVersionSeqNum())
	}
	require.NoError(m.t, os.WriteFile(filepath.Join(path, "db"), []byte(fmt.Sprintf("%d", migrations.CurrentDBVersionSeqNum())), 0644))
}

func (m *mockCentral) runMigrator(breakPoint string, forceRollback string) {
	dbm, err := Scan(m.mountPath, forceRollback)
	require.NoError(m.t, err)
	if breakPoint == breakAfterScan {
		return
	}
	replica, path, err := dbm.GetReplicaToMigrate()
	require.NoError(m.t, err)
	if breakPoint == breakAfterGetReplica {
		return
	}
	m.upgradeDB(path)
	if breakPoint == breakBeforePersist {
		return
	}

	require.NoError(m.t, dbm.Persist(replica))
	m.verifyDBVersion(migrations.CurrentPath(), migrations.CurrentDBVersionSeqNum())
	require.NoDirExists(m.t, filepath.Join(m.mountPath, restoreReplica))
}

func (m *mockCentral) runCentral() {
	require.NoError(m.t, migrations.SafeRemoveDBWithSymbolicLink(filepath.Join(m.mountPath, ".backup")))
	if version.CompareVersions(version.GetMainVersion(), "3.0.57.0") >= 0 {
		migrations.SetCurrent(migrations.CurrentPath())
	}
	if exists, _ := fileutils.Exists(filepath.Join(migrations.CurrentPath(), "db")); !exists {
		require.NoError(m.t, os.WriteFile(filepath.Join(migrations.CurrentPath(), "db"), []byte(fmt.Sprintf("%d", migrations.CurrentDBVersionSeqNum())), 0644))
	}

	m.verifyCurrent()
	require.NoDirExists(m.t, filepath.Join(m.mountPath, backupReplica))
}

func (m *mockCentral) restoreCentral(ver *versionPair, breakPoint string) {
	curVer := &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()}
	m.restore(ver)
	if breakPoint == "" {
		m.runMigrator(breakPoint, "")
	}
	m.runMigrator("", "")
	m.verifyReplica(backupReplica, curVer)
	m.runCentral()
}

func (m *mockCentral) rollbackCentral(ver *versionPair, breakpoint string, forceRollback string) {
	m.migrateWithVersion(ver, breakpoint, forceRollback)
	if breakpoint != "" {
		m.runMigrator("", "")
	}
	m.runCentral()
}

func (m *mockCentral) restore(ver *versionPair) {
	// Central should be in running state.
	m.verifyCurrent()

	restoreDir, err := os.MkdirTemp(migrations.DBMountPath(), ".restore-")
	require.NoError(m.t, os.Symlink(filepath.Base(restoreDir), filepath.Join(migrations.DBMountPath(), ".restore")))
	require.NoError(m.t, err)
	// backups from version lower than 3.0.57.0 do not have migration version.
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		m.setMigrationVersion(restoreDir, ver)
	}
}

func (m *mockCentral) verifyCurrent() {
	m.verifyReplica(currentReplica, &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()})
}

func (m *mockCentral) verifyReplica(replica string, ver *versionPair) {
	dbPath := filepath.Join(m.mountPath, replica)
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		m.verifyMigrationVersion(dbPath, ver)
	} else {
		require.NoFileExists(m.t, filepath.Join(dbPath, migrations.MigrationVersionFile))
		m.verifyMigrationVersion(dbPath, &versionPair{version: "0", seqNum: 0})
	}
	m.verifyDBVersion(dbPath, ver.seqNum)
}

func (m *mockCentral) setMigrationVersion(path string, ver *versionPair) {
	migVer := migrations.MigrationVersion{MainVersion: ver.version, SeqNum: ver.seqNum}
	bytes, err := yaml.Marshal(migVer)
	require.NoError(m.t, err)
	require.NoError(m.t, os.WriteFile(filepath.Join(path, migrations.MigrationVersionFile), bytes, 0644))
}

func (m *mockCentral) verifyMigrationVersion(dbPath string, ver *versionPair) {
	migVer, err := migrations.Read(dbPath)
	require.NoError(m.t, err)
	assert.Equal(m.t, ver.seqNum, migVer.SeqNum)
	require.Equal(m.t, ver.version, migVer.MainVersion)
}

func (m *mockCentral) verifyDBVersion(dbPath string, seqNum int) {
	bytes, err := os.ReadFile(filepath.Join(dbPath, "db"))
	require.NoError(m.t, err)
	dbSeq, err := strconv.Atoi(string(bytes))
	require.NoError(m.t, err)
	require.Equal(m.t, seqNum, dbSeq)
}

func (m *mockCentral) runMigratorWithBreaksInPersist(breakpoint string) {
	dbm, err := Scan(m.mountPath, "")
	require.NoError(m.t, err)
	replica, path, err := dbm.GetReplicaToMigrate()
	require.NoError(m.t, err)
	m.upgradeDB(path)
	// start to persist

	var prev string
	switch replica {
	case currentReplica:
		return
	case previousReplica:
		prev = ""
	case restoreReplica:
		prev = backupReplica
	case tempReplica:
		prev = previousReplica
	}
	_ = migrations.SafeRemoveDBWithSymbolicLink(prev)
	if breakpoint == breakAfterRemove {
		return
	}
	_ = fileutils.AtomicSymlink(prev, filepath.Join(m.mountPath, dbm.replicaMap[currentReplica].dirName))
	if breakpoint == breakBeforeCommitCurrent {
		return
	}
	_ = fileutils.AtomicSymlink(currentReplica, filepath.Join(m.mountPath, dbm.replicaMap[replica].dirName))
}
