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
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/version"
	vStore "github.com/stackrox/rox/pkg/version/postgres"
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
	t               *testing.T
	mountPath       string
	rollbackEnabled bool
	tp              *pgtest.TestPostgres
	// Set version function has to be provided by test itself.
	setVersion  func(t *testing.T, ver *versionPair)
	adminConfig *pgxpool.Config
	// May need to run both databases if testing case of upgrading from rocks version to a postgres version
	runBoth bool
}

// createCentral - creates a central that runs Rocks OR Postgres OR both.  Need to cover
// the condition where we have Rocks data that needs migrated to Postgres.  Need to test that
// we calculate the destination clone appropriately.
func createCentral(t *testing.T, runBoth bool) *mockCentral {
	mountDir, err := os.MkdirTemp("", "mock-central-")
	require.NoError(t, err)
	mock := mockCentral{t: t, mountPath: mountDir, runBoth: runBoth}

	if !features.PostgresDatastore.Enabled() || runBoth {
		dbPath := filepath.Join(mountDir, ".db-init")
		require.NoError(t, os.Mkdir(dbPath, 0755))
		require.NoError(t, os.Symlink(dbPath, filepath.Join(mountDir, "current")))
		migrationtestutils.SetDBMountPath(t, mountDir)
	}

	if features.PostgresDatastore.Enabled() {
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

	if features.PostgresDatastore.Enabled() {
		if m.rollbackEnabled && version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			m.verifyClonePostgres(postgres.PreviousClone, curVer)
		} else {
			assert.False(m.t, pgadmin.CheckIfDBExists(m.adminConfig, postgres.PreviousClone))
		}
	} else {
		if m.rollbackEnabled && version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			m.verifyClone(rocksdb.PreviousClone, curVer)
		} else {
			assert.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.PreviousClone))
		}
	}
}

func (m *mockCentral) upgradeDB(path, clone, pgClone string) {
	if features.PostgresDatastore.Enabled() {
		if pgadmin.CheckIfDBExists(m.adminConfig, pgClone) {
			pool := pgadmin.GetClonePool(m.adminConfig, pgClone)
			defer pool.Close()

			cloneVer, err := migrations.ReadVersionPostgres(pool)
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

func (m *mockCentral) runMigrator(breakPoint string, forceRollback string) {
	var dbm DBCloneManager

	if features.PostgresDatastore.Enabled() {
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
	if features.PostgresDatastore.Enabled() {
		require.NotEmpty(m.t, pgClone)
	}
	if !features.PostgresDatastore.Enabled() {
		require.NotEmpty(m.t, clone)
		require.NotEmpty(m.t, clonePath)
	}
	// If we are running rocks too, we need to either have just a pgClone OR both.
	if features.PostgresDatastore.Enabled() && m.runBoth {
		require.True(m.t, pgClone != "" || (pgClone != "" && clone != "" && clonePath != ""))
	}
	if breakPoint == breakAfterGetClone {
		return
	}

	m.upgradeDB(clonePath, clone, pgClone)
	if breakPoint == breakBeforePersist {
		return
	}

	persistBoth := false
	if clone != "" && pgClone != "" {
		persistBoth = true
	}

	require.NoError(m.t, dbm.Persist(clone, pgClone, persistBoth))

	if !features.PostgresDatastore.Enabled() {
		m.verifyDBVersion(migrations.CurrentPath(), migrations.CurrentDBVersionSeqNum())
		require.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.RestoreClone))
	}
}

func (m *mockCentral) runCentral() {
	if !features.PostgresDatastore.Enabled() || m.runBoth {
		require.NoError(m.t, migrations.SafeRemoveDBWithSymbolicLink(filepath.Join(m.mountPath, ".backup")))
		if version.CompareVersions(version.GetMainVersion(), "3.0.57.0") >= 0 {
			migrations.SetCurrent(migrations.CurrentPath())
		}
		if exists, _ := fileutils.Exists(filepath.Join(migrations.CurrentPath(), "db")); !exists {
			require.NoError(m.t, os.WriteFile(filepath.Join(migrations.CurrentPath(), "db"), []byte(fmt.Sprintf("%d", migrations.CurrentDBVersionSeqNum())), 0644))
		}
	}

	if features.PostgresDatastore.Enabled() {
		if version.CompareVersions(version.GetMainVersion(), "3.0.57.0") >= 0 {
			pool := pgadmin.GetClonePool(m.adminConfig, migrations.GetCurrentClone())

			migrations.SetCurrentVersionPostgres(pool)
			pool.Close()
		}
	}
	m.verifyCurrent()

	if !features.PostgresDatastore.Enabled() || m.runBoth {
		require.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.BackupClone))
	}
}

func (m *mockCentral) restoreCentral(ver *versionPair, breakPoint string) {
	curVer := &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()}
	m.restore(ver)
	if breakPoint == "" {
		m.runMigrator(breakPoint, "")
	}
	m.runMigrator("", "")
	if features.PostgresDatastore.Enabled() {
		m.verifyClonePostgres(postgres.BackupClone, curVer)
		m.runCentral()
	} else {
		m.verifyClone(rocksdb.BackupClone, curVer)
		m.runCentral()
	}
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

	if features.PostgresDatastore.Enabled() {
		restoreDB := migrations.RestoreDatabase
		pgtest.CreateDatabase(m.t, restoreDB)

		// backups from version lower than 3.0.57.0 do not have migration version.
		if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
			m.setMigrationVersionPostgres(restoreDB, ver)
		}
	} else {
		restoreDir, err := os.MkdirTemp(migrations.DBMountPath(), ".restore-")
		require.NoError(m.t, os.Symlink(filepath.Base(restoreDir), filepath.Join(migrations.DBMountPath(), ".restore")))
		require.NoError(m.t, err)
		// backups from version lower than 3.0.57.0 do not have migration version.
		if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
			m.setMigrationVersion(restoreDir, ver)
		}
	}
}

func (m *mockCentral) verifyCurrent() {
	if features.PostgresDatastore.Enabled() {
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
	pool := pgadmin.GetClonePool(m.tp.Config(), clone)
	defer pool.Close()

	ctx := sac.WithAllAccess(context.Background())
	store := vStore.New(ctx, pool)

	err := store.Upsert(ctx, &storage.Version{SeqNum: int32(ver.seqNum), Version: ver.version})
	require.NoError(m.t, err)

}

func (m *mockCentral) verifyMigrationVersion(dbPath string, ver *versionPair) {
	migVer, err := migrations.Read(dbPath)
	require.NoError(m.t, err)
	assert.Equal(m.t, ver.seqNum, migVer.SeqNum)
	require.Equal(m.t, ver.version, migVer.MainVersion)
}

func (m *mockCentral) verifyMigrationVersionPostgres(clone string, ver *versionPair) {
	pool := pgadmin.GetClonePool(m.adminConfig, clone)
	defer pool.Close()

	migVer, err := migrations.ReadVersionPostgres(pool)
	require.NoError(m.t, err)
	log.Infof("clone => %q.  Version incoming => %v", clone, ver)
	assert.Equal(m.t, ver.seqNum, migVer.SeqNum)
	require.Equal(m.t, ver.version, migVer.MainVersion)
}

func (m *mockCentral) getCloneVersion(clone string) (*migrations.MigrationVersion, error) {
	var pool *pgxpool.Pool
	if clone == migrations.GetCurrentClone() {
		pool = m.tp.Pool
	} else {
		pool = pgtest.ForTCustomPool(m.t, clone)
	}
	defer pool.Close()

	ctx := sac.WithAllAccess(context.Background())

	store := vStore.New(ctx, pool)

	version, exists, err := store.Get(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		return &migrations.MigrationVersion{MainVersion: "0", SeqNum: 0}, nil
	}

	return &migrations.MigrationVersion{MainVersion: version.Version, SeqNum: int(version.SeqNum)}, nil
}

func (m *mockCentral) verifyDBVersion(dbPath string, seqNum int) {
	bytes, err := os.ReadFile(filepath.Join(dbPath, "db"))
	require.NoError(m.t, err)
	dbSeq, err := strconv.Atoi(string(bytes))
	require.NoError(m.t, err)
	require.Equal(m.t, seqNum, dbSeq)
}

func (m *mockCentral) runMigratorWithBreaksInPersist(breakpoint string) {
	if features.PostgresDatastore.Enabled() {
		source := pgtest.GetConnectionString(m.t)
		sourceMap, _ := pgconfig.ParseSource(source)
		config, err := pgxpool.ParseConfig(source)
		require.NoError(m.t, err)

		dbm := postgres.New("", config, sourceMap)
		err = dbm.Scan()
		require.NoError(m.t, err)
		clone, _, err := dbm.GetCloneToMigrate(nil)
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
		_ = migrations.SafeRemoveDBWithSymbolicLink(prev)
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
