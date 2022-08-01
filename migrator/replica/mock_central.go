package replica

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/replica/postgres"
	"github.com/stackrox/rox/migrator/replica/rocksdb"
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
	breakAfterGetReplica     = "get-replica"
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

func createCentralPostgres(t *testing.T) *mockCentral {
	mock := mockCentral{t: t}

	mock.tp = pgtest.ForTCustomDB(t, "postgres")
	err := pgtest.CreateDatabase(t, "central_active")
	require.NoError(t, err)

	source := pgtest.GetConnectionString(mock.t)
	mock.adminConfig, _ = pgxpool.ParseConfig(source)

	return &mock
}

func (m *mockCentral) destroyCentral() {
	if m.tp != nil {
		pgtest.DropDatabase(m.t, "central_active")
		pgtest.DropDatabase(m.t, "central_previous")
		pgtest.DropDatabase(m.t, "central_backup")
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
	if features.PostgresDatastore.Enabled() {
		m.runCentralPostgres()
	} else {
		m.runCentral()
	}
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

	if features.PostgresDatastore.Enabled() {
		m.runCentralPostgres()
		if m.rollbackEnabled && version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			m.verifyReplica(postgres.PreviousReplica, curVer)
		} else {
			assert.False(m.t, pgadmin.CheckIfDBExists(m.adminConfig, postgres.PreviousReplica))
		}
	} else {
		m.runCentral()
		if m.rollbackEnabled && version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			m.verifyReplica(rocksdb.PreviousReplica, curVer)
		} else {
			assert.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.PreviousReplica))
		}
	}
}

func (m *mockCentral) upgradeDB(path, replica string) {
	if features.PostgresDatastore.Enabled() {
		if pgadmin.CheckIfDBExists(m.adminConfig, replica) {
			pool := pgadmin.GetReplicaPool(m.adminConfig, replica)
			defer pool.Close()

			replicaVer, err := migrations.ReadVersion(pool)
			require.NoError(m.t, err)
			require.LessOrEqual(m.t, replicaVer.SeqNum, migrations.CurrentDBVersionSeqNum())
		}
		return
	}

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
	if features.PostgresDatastore.Enabled() {
		source := pgtest.GetConnectionString(m.t)
		sourceMap, _ := pgconfig.ParseSource(source)
		config, err := pgxpool.ParseConfig(source)
		require.NoError(m.t, err)

		dbm := postgres.New(forceRollback, config, sourceMap)
		err = dbm.Scan()
		require.NoError(m.t, err)
		if breakPoint == breakAfterScan {
			return
		}
		replica, _, err := dbm.GetReplicaToMigrate()
		require.NoError(m.t, err)
		if breakPoint == breakAfterGetReplica {
			return
		}
		m.upgradeDB("", replica)
		if breakPoint == breakBeforePersist {
			return
		}

		require.NoError(m.t, dbm.Persist(replica))
	} else {
		dbm := rocksdb.New(m.mountPath, forceRollback)
		err := dbm.Scan()
		require.NoError(m.t, err)
		if breakPoint == breakAfterScan {
			return
		}
		replica, path, err := dbm.GetReplicaToMigrate()
		require.NoError(m.t, err)
		if breakPoint == breakAfterGetReplica {
			return
		}
		m.upgradeDB(path, "")
		if breakPoint == breakBeforePersist {
			return
		}

		require.NoError(m.t, dbm.Persist(replica))
		m.verifyDBVersion(migrations.CurrentPath(), migrations.CurrentDBVersionSeqNum())
		require.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.RestoreReplica))
	}
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
	require.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.BackupReplica))

}

func (m *mockCentral) runCentralPostgres() {
	if version.CompareVersions(version.GetMainVersion(), "3.0.57.0") >= 0 {
		pool := pgadmin.GetReplicaPool(m.adminConfig, "central_active")

		migrations.SetCurrentVersion(pool)
		pool.Close()
	}

	m.verifyCurrent()
}

func (m *mockCentral) restoreCentral(ver *versionPair, breakPoint string) {
	curVer := &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()}
	m.restore(ver)
	if breakPoint == "" {
		m.runMigrator(breakPoint, "")
	}
	m.runMigrator("", "")
	if features.PostgresDatastore.Enabled() {
		m.verifyReplica(postgres.BackupReplica, curVer)
		m.runCentralPostgres()
	} else {
		m.verifyReplica(rocksdb.BackupReplica, curVer)
		m.runCentral()
	}
}

func (m *mockCentral) rollbackCentral(ver *versionPair, breakpoint string, forceRollback string) {
	m.migrateWithVersion(ver, breakpoint, forceRollback)
	if breakpoint != "" {
		m.runMigrator("", "")
	}
	if features.PostgresDatastore.Enabled() {
		m.runCentralPostgres()
	} else {
		m.runCentral()
	}
}

func (m *mockCentral) restore(ver *versionPair) {
	// Central should be in running state.
	m.verifyCurrent()

	if features.PostgresDatastore.Enabled() {
		restoreDB := "central_restore"
		err := pgtest.CreateDatabase(m.t, restoreDB)
		require.NoError(m.t, err)

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
		m.verifyReplica(postgres.CurrentReplica, &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()})
	} else {
		m.verifyReplica(rocksdb.CurrentReplica, &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum()})
	}
}

func (m *mockCentral) verifyReplica(replica string, ver *versionPair) {
	dbPath := filepath.Join(m.mountPath, replica)
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		if features.PostgresDatastore.Enabled() {
			m.verifyMigrationVersionPostgres(replica, ver)
		} else {
			m.verifyMigrationVersion(dbPath, ver)
		}
	} else {
		if features.PostgresDatastore.Enabled() {
			m.verifyMigrationVersionPostgres(replica, &versionPair{version: "0", seqNum: 0})
		} else {
			require.NoFileExists(m.t, filepath.Join(dbPath, migrations.MigrationVersionFile))
			m.verifyMigrationVersion(dbPath, &versionPair{version: "0", seqNum: 0})
		}
	}

	// For Postgres performing this verification in verifyMigrationVersionPostgres
	if !features.PostgresDatastore.Enabled() {
		m.verifyDBVersion(dbPath, ver.seqNum)
	}
}

func (m *mockCentral) setMigrationVersion(path string, ver *versionPair) {
	migVer := migrations.MigrationVersion{MainVersion: ver.version, SeqNum: ver.seqNum}
	bytes, err := yaml.Marshal(migVer)
	require.NoError(m.t, err)
	require.NoError(m.t, os.WriteFile(filepath.Join(path, migrations.MigrationVersionFile), bytes, 0644))
}

func (m *mockCentral) setMigrationVersionPostgres(replica string, ver *versionPair) {
	pool := pgadmin.GetReplicaPool(m.tp.Config(), replica)
	defer pool.Close()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Version)))
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

func (m *mockCentral) verifyMigrationVersionPostgres(replica string, ver *versionPair) {
	pool := pgadmin.GetReplicaPool(m.adminConfig, replica)
	defer pool.Close()

	migVer, err := migrations.ReadVersion(pool)
	require.NoError(m.t, err)
	assert.Equal(m.t, ver.seqNum, migVer.SeqNum)
	require.Equal(m.t, ver.version, migVer.MainVersion)
}

func (m *mockCentral) getReplicaVersion(replica string) (*migrations.MigrationVersion, error) {
	var pool *pgxpool.Pool
	if replica == migrations.CurrentReplica() {
		pool = m.tp.Pool
	} else {
		pool = pgtest.ForTCustomPool(m.t, replica)
	}
	defer pool.Close()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Version)))

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
		replica, _, err := dbm.GetReplicaToMigrate()
		require.NoError(m.t, err)
		m.upgradeDB("", replica)

		var prev string
		switch replica {
		case postgres.CurrentReplica:
			return
		case postgres.PreviousReplica:
			prev = ""
		case postgres.RestoreReplica:
			prev = postgres.BackupReplica
		case postgres.TempReplica:
			prev = postgres.PreviousReplica
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
		_ = pgadmin.RenameDB(connectPool, prev, postgres.CurrentReplica)
		if breakpoint == breakBeforeCommitCurrent {
			return
		}
		_ = pgadmin.RenameDB(connectPool, postgres.CurrentReplica, replica)
	} else {
		dbm := rocksdb.New(m.mountPath, "")
		err := dbm.Scan()
		require.NoError(m.t, err)
		replica, path, err := dbm.GetReplicaToMigrate()
		require.NoError(m.t, err)
		m.upgradeDB(path, "")
		// start to persist

		var prev string
		switch replica {
		case rocksdb.CurrentReplica:
			return
		case rocksdb.PreviousReplica:
			prev = ""
		case rocksdb.RestoreReplica:
			prev = rocksdb.BackupReplica
		case rocksdb.TempReplica:
			prev = rocksdb.PreviousReplica
		}
		_ = migrations.SafeRemoveDBWithSymbolicLink(prev)
		if breakpoint == breakAfterRemove {
			return
		}
		_ = fileutils.AtomicSymlink(prev, filepath.Join(m.mountPath, dbm.ReplicaMap[rocksdb.CurrentReplica].GetDirName()))
		if breakpoint == breakBeforeCommitCurrent {
			return
		}
		_ = fileutils.AtomicSymlink(rocksdb.CurrentReplica, filepath.Join(m.mountPath, dbm.ReplicaMap[replica].GetDirName()))
	}
}
