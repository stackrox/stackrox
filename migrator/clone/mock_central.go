package clone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/clone/postgres"
	"github.com/stackrox/rox/migrator/clone/rocksdb"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	breakAfterScan           = "scan"
	breakAfterGetClone       = "get-clone"
	breakBeforePersist       = "persist"
	breakAfterRemove         = "remove"
	breakBeforeCommitCurrent = "current"
	breakBeforeCleanUp       = "cleanup"
)

var (
	log = logging.CurrentModule().Logger()
)

type versionPair struct {
	version string
	seqNum  int
}

type mockCentral struct {
	t         *testing.T
	ctx       context.Context
	mountPath string
	tp        *pgtest.TestPostgres
	// Set version function has to be provided by test itself.
	setVersion  func(t *testing.T, ver *versionPair)
	adminConfig *pgxpool.Config
	// May need to run both databases if testing case of upgrading from rocks version to a postgres version
	runBoth    bool
	updateBoth bool
	gc         migGorm.Config
}

// createCentral - creates a central that runs Rocks OR Postgres OR both.  Need to cover
// the condition where we have Rocks data that needs migrated to Postgres.  Need to test that
// we calculate the destination clone appropriately.
func createCentral(t *testing.T, runBoth bool) *mockCentral {
	// Initialize mock test config
	_ = migGorm.SetupAndGetMockConfig(t)

	mountDir, err := os.MkdirTemp("", "mock-central-")
	require.NoError(t, err)
	mock := mockCentral{t: t, mountPath: mountDir, runBoth: runBoth}
	mock.ctx = sac.WithAllAccess(context.Background())

	if !env.PostgresDatastoreEnabled.BooleanSetting() || runBoth {
		dbPath := filepath.Join(mountDir, ".db-init")
		require.NoError(t, os.Mkdir(dbPath, 0755))
		require.NoError(t, os.Symlink(dbPath, filepath.Join(mountDir, "current")))
		migrationtestutils.SetDBMountPath(t, mountDir)
	}

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		mock.tp = pgtest.ForTCustomDB(t, "postgres")
		pgtest.CreateDatabase(t, migrations.GetCurrentClone())

		source := pgtest.GetConnectionString(mock.t)
		mock.adminConfig, _ = pgxpool.ParseConfig(source)
	}
	return &mock
}

func (m *mockCentral) destroyCentral() {
	if m.tp != nil {
		pgtest.DropDatabase(m.t, migrations.GetCurrentClone())
		pgtest.DropDatabase(m.t, migrations.GetPreviousClone())
		pgtest.DropDatabase(m.t, migrations.GetBackupClone())
		m.tp.Teardown(m.t)
	}
	_ = os.RemoveAll(m.mountPath)
}

func (m *mockCentral) rebootCentral() {
	curSeq := migrations.CurrentDBVersionSeqNum()
	curVer := version.GetMainVersion()
	m.runMigrator("", "", false)
	m.runCentral()
	assert.Equal(m.t, curSeq, migrations.CurrentDBVersionSeqNum())
	assert.Equal(m.t, curVer, version.GetMainVersion())
}

func (m *mockCentral) migrateWithVersion(ver *versionPair, breakpoint string, forceRollback string) {
	m.setVersion(m.t, ver)
	m.runMigrator(breakpoint, forceRollback, false)
}

func (m *mockCentral) upgradeCentral(ver *versionPair, breakpoint string) {
	curVer := &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()}
	m.migrateWithVersion(ver, breakpoint, "")
	// Re-run migrator if the previous one breaks
	if breakpoint != "" {
		m.runMigrator("", "", false)
	}

	m.runCentral()

	if env.PostgresDatastoreEnabled.BooleanSetting() && m.runBoth {
		if version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			if pgadmin.CheckIfDBExists(m.adminConfig, postgres.TempClone) {
				m.verifyClonePostgres(postgres.TempClone, curVer)
			}
		} else {
			assert.False(m.t, pgadmin.CheckIfDBExists(m.adminConfig, postgres.TempClone))
		}
	} else if env.PostgresDatastoreEnabled.BooleanSetting() {
		if version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			m.verifyClonePostgres(postgres.PreviousClone, curVer)
		} else {
			assert.False(m.t, pgadmin.CheckIfDBExists(m.adminConfig, postgres.PreviousClone))
		}
	} else {
		if version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			m.verifyClone(rocksdb.PreviousClone, curVer)
		} else {
			assert.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.PreviousClone))
		}
	}
}

func (m *mockCentral) upgradeDB(path, clone, pgClone string) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		if pgadmin.CheckIfDBExists(m.adminConfig, pgClone) {
			cloneVer, err := migVer.ReadVersionPostgres(m.ctx, pgClone)
			require.NoError(m.t, err)
			require.LessOrEqual(m.t, cloneVer.SeqNum, migrations.CurrentDBVersionSeqNum())
		}
		if !m.runBoth {
			return
		}
	}
	if path != "" {
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
}

func (m *mockCentral) runMigrator(breakPoint string, forceRollback string, unsupportedRocks bool) {
	var dbm DBCloneManager

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		source := pgtest.GetConnectionString(m.t)
		sourceMap, _ := pgconfig.ParseSource(source)
		config, err := pgxpool.ParseConfig(source)
		require.NoError(m.t, err)

		dbm = NewPostgres(m.mountPath, forceRollback, config, sourceMap)
	} else {
		dbm = New(m.mountPath, forceRollback)
	}

	err := dbm.Scan()
	if err != nil {
		log.Info(err)
	}
	require.NoError(m.t, err)
	if breakPoint == breakAfterScan {
		return
	}

	clone, clonePath, pgClone, err := dbm.GetCloneToMigrate()
	require.NoError(m.t, err)
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		require.NotEmpty(m.t, pgClone)
	}
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		require.NotEmpty(m.t, clone)
		require.NotEmpty(m.t, clonePath)
	}
	// If we are running rocks too, we need to either have just a pgClone OR both.
	if env.PostgresDatastoreEnabled.BooleanSetting() && m.runBoth {
		require.True(m.t, pgClone != "" || (pgClone != "" && clone != "" && clonePath != ""))
	}
	if breakPoint == breakAfterGetClone {
		return
	}

	m.upgradeDB(clonePath, clone, pgClone)
	if breakPoint == breakBeforePersist {
		return
	}

	// assume we only need to persist one.
	m.updateBoth = false
	if clone != "" && pgClone != "" {
		m.updateBoth = true

		// If we are migrating from Rocks, it could be a subsequent upgrade.  If so we will
		// have deleted the current Postgres DB.  We need to re-create it here for the rest
		// of the test.  This will naturally be recreated in migrator, but we are focused on the clones
		// here and such that code is not executed as part of this test
		pgtest.CreateDatabase(m.t, pgClone)
	}

	require.NoError(m.t, dbm.Persist(clone, pgClone, m.updateBoth))

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		m.verifyDBVersion(migrations.CurrentPath(), migrations.CurrentDBVersionSeqNum())
		require.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.RestoreClone))
	}
}

func (m *mockCentral) runCentral() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() || m.updateBoth {
		require.NoError(m.t, migrations.SafeRemoveDBWithSymbolicLink(filepath.Join(m.mountPath, ".backup")))
		if version.CompareVersions(version.GetMainVersion(), "3.0.57.0") >= 0 {
			migrations.SetCurrent(migrations.CurrentPath())
		}
		if exists, _ := fileutils.Exists(filepath.Join(migrations.CurrentPath(), "db")); !exists {
			require.NoError(m.t, os.WriteFile(filepath.Join(migrations.CurrentPath(), "db"), []byte(fmt.Sprintf("%d", migrations.CurrentDBVersionSeqNum())), 0644))
		}
	}

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		if version.CompareVersions(version.GetMainVersion(), "3.0.57.0") >= 0 {
			migVer.SetCurrentVersionPostgres(m.ctx)
		}
	}
	m.verifyCurrent()

	if !env.PostgresDatastoreEnabled.BooleanSetting() || m.runBoth {
		require.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.BackupClone))
	}
}

func (m *mockCentral) restoreCentral(ver *versionPair, breakPoint string, rocksToPostgres bool) {
	curVer := &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()}
	m.restore(ver, rocksToPostgres)
	if breakPoint == "" {
		m.runMigrator(breakPoint, "", false)
	}
	m.runMigrator("", "", false)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		m.verifyClonePostgres(postgres.BackupClone, curVer)
	} else {
		m.verifyClone(rocksdb.BackupClone, curVer)
	}

	m.runCentral()
}

func (m *mockCentral) rollbackCentral(ver *versionPair, breakpoint string, forceRollback string) {
	m.migrateWithVersion(ver, breakpoint, forceRollback)
	if breakpoint != "" {
		m.runMigrator("", "", false)
	}

	m.runCentral()
}

func (m *mockCentral) restore(ver *versionPair, rocksToPostgres bool) {
	// Central should be in running state.
	m.verifyCurrent()

	if !env.PostgresDatastoreEnabled.BooleanSetting() || rocksToPostgres {
		restoreDir, err := os.MkdirTemp(migrations.DBMountPath(), ".restore-")
		require.NoError(m.t, os.Symlink(filepath.Base(restoreDir), filepath.Join(migrations.DBMountPath(), ".restore")))
		require.NoError(m.t, err)
		// backups from version lower than 3.0.57.0 do not have migration version.
		if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
			m.setMigrationVersion(restoreDir, ver)
		}
	}

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		pgtest.CreateDatabase(m.t, migrations.RestoreDatabase)

		// backups from version lower than 3.0.57.0 do not have migration version.
		if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
			m.setMigrationVersionPostgres(migrations.RestoreDatabase, ver)
		}
	}
}

func (m *mockCentral) verifyCurrent() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		m.verifyClonePostgres(postgres.CurrentClone, &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()})
	} else {
		m.verifyClone(rocksdb.CurrentClone, &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()})
	}
}

func (m *mockCentral) verifyClone(clone string, ver *versionPair) {
	dbPath := filepath.Join(m.mountPath, clone)
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		m.verifyMigrationVersion(dbPath, ver)
	} else {
		require.NoFileExists(m.t, filepath.Join(dbPath, migrations.MigrationVersionFile))
		m.verifyMigrationVersion(dbPath, &versionPair{version: "0", seqNum: 0})
	}
	m.verifyDBVersion(dbPath, ver.seqNum)
}

func (m *mockCentral) verifyClonePostgres(clone string, ver *versionPair) {
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		m.verifyMigrationVersionPostgres(clone, ver)
	} else {
		m.verifyMigrationVersionPostgres(clone, &versionPair{version: "0", seqNum: 0})
	}
}

func (m *mockCentral) setMigrationVersion(path string, ver *versionPair) {
	migVer := migrations.MigrationVersion{MainVersion: ver.version, SeqNum: ver.seqNum}
	bytes, err := yaml.Marshal(migVer)
	require.NoError(m.t, err)
	require.NoError(m.t, os.WriteFile(filepath.Join(path, migrations.MigrationVersionFile), bytes, 0644))
}

func (m *mockCentral) setMigrationVersionPostgres(clone string, ver *versionPair) {
	migVer.SetVersionPostgres(m.ctx, clone, &storage.Version{SeqNum: int32(ver.seqNum), Version: ver.version})
}

func (m *mockCentral) verifyMigrationVersion(dbPath string, ver *versionPair) {
	migVer, err := migrations.Read(dbPath)
	require.NoError(m.t, err)
	assert.Equal(m.t, ver.seqNum, migVer.SeqNum)
	require.Equal(m.t, ver.version, migVer.MainVersion)
}

func (m *mockCentral) verifyMigrationVersionPostgres(clone string, ver *versionPair) {
	migVer, err := migVer.ReadVersionPostgres(m.ctx, clone)
	require.NoError(m.t, err)
	log.Infof("clone => %q.  Version incoming => %v", clone, ver)
	assert.Equal(m.t, ver.seqNum, migVer.SeqNum)
	require.Equal(m.t, ver.version, migVer.MainVersion)
}

func (m *mockCentral) getCloneVersion(clone string) (*migrations.MigrationVersion, error) {
	return migVer.ReadVersionPostgres(m.ctx, clone)
}

func (m *mockCentral) verifyDBVersion(dbPath string, seqNum int) {
	bytes, err := os.ReadFile(filepath.Join(dbPath, "db"))
	require.NoError(m.t, err)
	dbSeq, err := strconv.Atoi(string(bytes))
	require.NoError(m.t, err)
	require.Equal(m.t, seqNum, dbSeq)
}

func (m *mockCentral) runMigratorWithBreaksInPersist(breakpoint string) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		source := pgtest.GetConnectionString(m.t)
		sourceMap, _ := pgconfig.ParseSource(source)
		config, err := pgxpool.ParseConfig(source)
		require.NoError(m.t, err)

		dbm := postgres.New("", config, sourceMap)
		err = dbm.Scan()
		require.NoError(m.t, err)
		clone, _, err := dbm.GetCloneToMigrate(nil, false)
		require.NoError(m.t, err)
		m.upgradeDB("", "", clone)

		var prev string
		switch clone {
		case postgres.CurrentClone:
			return
		case postgres.PreviousClone:
			prev = ""
		case postgres.RestoreClone:
			prev = postgres.BackupClone
		case postgres.TempClone:
			prev = postgres.PreviousClone
		}

		// Connect to different database for admin functions
		connectPool := pgadmin.GetAdminPool(m.adminConfig)
		// Close the admin connection pool
		defer connectPool.Close()

		log.Infof("runMigratorWithBreaksInPersist, prev = %s", prev)
		pgtest.DropDatabase(m.t, prev)
		if breakpoint == breakAfterRemove {
			return
		}
		_ = pgadmin.RenameDB(connectPool, prev, postgres.CurrentClone)
		if breakpoint == breakBeforeCommitCurrent {
			return
		}
		_ = pgadmin.RenameDB(connectPool, postgres.CurrentClone, clone)
	} else {
		dbm := rocksdb.New(m.mountPath, "")
		err := dbm.Scan()
		require.NoError(m.t, err)
		clone, path, err := dbm.GetCloneToMigrate()
		require.NoError(m.t, err)
		m.upgradeDB(path, "", "")
		// start to persist

		var prev string
		switch clone {
		case rocksdb.CurrentClone:
			return
		case rocksdb.PreviousClone:
			prev = ""
		case rocksdb.RestoreClone:
			prev = rocksdb.BackupClone
		case rocksdb.TempClone:
			prev = rocksdb.PreviousClone
		}
		_ = migrations.SafeRemoveDBWithSymbolicLink(filepath.Join(migrations.DBMountPath(), prev))
		if breakpoint == breakAfterRemove {
			return
		}
		_ = fileutils.AtomicSymlink(prev, filepath.Join(m.mountPath, dbm.GetDirName(rocksdb.CurrentClone)))
		if breakpoint == breakBeforeCommitCurrent {
			return
		}
		_ = fileutils.AtomicSymlink(rocksdb.CurrentClone, filepath.Join(m.mountPath, dbm.GetDirName(clone)))
	}
}

func (m *mockCentral) removePreviousClone() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		pgtest.DropDatabase(m.t, postgres.PreviousClone)
	} else {
		_ = migrations.SafeRemoveDBWithSymbolicLink(filepath.Join(migrations.DBMountPath(), rocksdb.PreviousClone))
	}
}
