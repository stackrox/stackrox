package clone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	pgClone "github.com/stackrox/rox/migrator/clone/postgres"
	"github.com/stackrox/rox/migrator/clone/rocksdb"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	"github.com/stackrox/rox/migrator/postgreshelper"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres"
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
	version   string
	seqNum    int
	minSeqNum int
}

type mockCentral struct {
	t         *testing.T
	ctx       context.Context
	mountPath string
	tp        *pgtest.TestPostgres
	// Set version function has to be provided by test itself.
	setVersion  func(t *testing.T, ver *versionPair)
	adminConfig *postgres.Config
	// May need to run both databases if testing case of upgrading from rocks version to a postgres version
	runBoth bool
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

	if runBoth {
		dbPath := filepath.Join(mountDir, ".db-init")
		require.NoError(t, os.Mkdir(dbPath, 0755))
		require.NoError(t, os.Symlink(dbPath, filepath.Join(mountDir, "current")))
		migrationtestutils.SetDBMountPath(t, mountDir)
	}

	mock.tp = pgtest.ForTCustomDB(t, "postgres")
	pgtest.CreateDatabase(t, migrations.GetCurrentClone())

	source := pgtest.GetConnectionString(mock.t)
	mock.adminConfig, _ = postgres.ParseConfig(source)
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

func (m *mockCentral) rebootCentral() error {
	curSeq := migrations.CurrentDBVersionSeqNum()
	curVer := version.GetMainVersion()
	curMinSeq := migrations.MinimumSupportedDBVersionSeqNum()
	if err := m.runMigrator("", ""); err != nil {
		return err
	}
	m.runCentral()
	assert.Equal(m.t, curSeq, migrations.CurrentDBVersionSeqNum())
	assert.Equal(m.t, curVer, version.GetMainVersion())
	assert.Equal(m.t, curMinSeq, migrations.MinimumSupportedDBVersionSeqNum())

	return nil
}

func (m *mockCentral) migrateWithVersion(ver *versionPair, breakpoint string, forceRollback string) error {
	m.setVersion(m.t, ver)
	return m.runMigrator(breakpoint, forceRollback)
}

// legacyUpgrade emulates the legacy database upgrade.
func (m *mockCentral) legacyUpgrade(t *testing.T, ver *versionPair, previousVer *versionPair) {
	path := filepath.Join(m.mountPath, rocksdb.CurrentClone)
	require.NoError(m.t, os.WriteFile(filepath.Join(path, "db"), []byte(fmt.Sprintf("%d", ver.seqNum)), 0644))

	m.setMigrationVersion(path, ver)

	if previousVer != nil {
		previousDir, err := os.MkdirTemp(migrations.DBMountPath(), ".previous-")
		require.NoError(t, err)
		require.NoError(m.t, os.Symlink(filepath.Base(previousDir), filepath.Join(migrations.DBMountPath(), ".previous")))

		prevPath := filepath.Join(m.mountPath, rocksdb.PreviousClone)
		require.NoError(m.t, os.WriteFile(filepath.Join(prevPath, "db"), []byte(fmt.Sprintf("%d", previousVer.seqNum)), 0644))

		m.setMigrationVersion(prevPath, previousVer)
	}
}

func (m *mockCentral) upgradeCentral(ver *versionPair, breakpoint string) error {
	curVer := &versionPair{
		version:   version.GetMainVersion(),
		seqNum:    migrations.CurrentDBVersionSeqNum(),
		minSeqNum: migrations.MinimumSupportedDBVersionSeqNum()}

	if err := m.migrateWithVersion(ver, breakpoint, ""); err != nil {
		return err
	}
	// Re-run migrator if the previous one breaks
	if breakpoint != "" {
		if err := m.runMigrator("", ""); err != nil {
			return err
		}
	}

	m.runCentral()

	if m.runBoth {
		if version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			if exists, _ := pgadmin.CheckIfDBExists(m.adminConfig, pgClone.TempClone); exists {
				m.verifyClonePostgres(pgClone.TempClone, curVer)
			}
		} else {
			exists, err := pgadmin.CheckIfDBExists(m.adminConfig, pgClone.TempClone)
			assert.NoError(m.t, err)
			assert.False(m.t, exists)
		}
	} else {
		if version.CompareVersions(curVer.version, "3.0.57.0") >= 0 {
			m.verifyClonePostgres(pgClone.PreviousClone, curVer)
		} else {
			exists, err := pgadmin.CheckIfDBExists(m.adminConfig, pgClone.PreviousClone)
			assert.NoError(m.t, err)
			assert.False(m.t, exists)
		}
	}

	return nil
}

func (m *mockCentral) upgradeDB(pgClone string) {
	if exists, _ := pgadmin.CheckIfDBExists(m.adminConfig, pgClone); exists {
		cloneVer, err := migVer.ReadVersionPostgres(m.ctx, pgClone)
		require.NoError(m.t, err)
		require.LessOrEqual(m.t, cloneVer.SeqNum, migrations.CurrentDBVersionSeqNum())
	}
	if !m.runBoth {
		return
	}
}

func (m *mockCentral) downgradeDB(pgClone string) {
	if exists, _ := pgadmin.CheckIfDBExists(m.adminConfig, pgClone); exists {
		cloneVer, err := migVer.ReadVersionPostgres(m.ctx, pgClone)
		require.NoError(m.t, err)
		require.GreaterOrEqual(m.t, cloneVer.SeqNum, migrations.CurrentDBVersionSeqNum())
	}
	if !m.runBoth {
		return
	}
}

func (m *mockCentral) runMigrator(breakPoint string, forceRollback string) error {
	source := pgtest.GetConnectionString(m.t)
	sourceMap, _ := pgconfig.ParseSource(source)
	config, err := postgres.ParseConfig(source)
	require.NoError(m.t, err)

	dbm := NewPostgres(m.mountPath, forceRollback, config, sourceMap)

	err = dbm.Scan()
	if err != nil {
		log.Info(err)
	}
	require.NoError(m.t, err)
	if breakPoint == breakAfterScan {
		return nil
	}

	pgClone, err := dbm.GetCloneToMigrate()
	log.Infof("SHREWS -- %v", pgClone)
	log.Infof("SHREWS -- %v", err)
	if err != nil {
		return err
	}
	require.NotEmpty(m.t, pgClone)

	// If we are running rocks too, we need to either have just a pgClone OR both.
	if m.runBoth {
		log.Infof("SHREWS run both -- %v", pgClone)
		require.True(m.t, pgClone != "")
	}
	if breakPoint == breakAfterGetClone {
		return nil
	}

	if forceRollback == "" {
		m.upgradeDB(pgClone)
	} else {
		m.downgradeDB(pgClone)
	}
	if breakPoint == breakBeforePersist {
		return nil
	}

	require.NoError(m.t, dbm.Persist(pgClone))

	return nil
}

func (m *mockCentral) runCentral() {
	if version.CompareVersions(version.GetMainVersion(), "3.0.57.0") >= 0 {
		migVer.SetCurrentVersionPostgres(m.ctx)
	}
	m.verifyCurrent()

	if m.runBoth {
		require.NoDirExists(m.t, filepath.Join(m.mountPath, rocksdb.BackupClone))
	}
}

func (m *mockCentral) restoreCentral(ver *versionPair, breakPoint string, rocksToPostgres bool) error {
	curVer := &versionPair{
		version:   version.GetMainVersion(),
		seqNum:    migrations.CurrentDBVersionSeqNum(),
		minSeqNum: migrations.MinimumSupportedDBVersionSeqNum(),
	}
	m.restore(ver, rocksToPostgres)
	if breakPoint == "" {
		if err := m.runMigrator(breakPoint, ""); err != nil {
			return err
		}
	}
	if err := m.runMigrator("", ""); err != nil {
		return err
	}

	m.verifyClonePostgres(pgClone.BackupClone, curVer)

	m.runCentral()

	return nil
}

func (m *mockCentral) rollbackCentral(ver *versionPair, breakpoint string, forceRollback string) error {
	if err := m.migrateWithVersion(ver, breakpoint, forceRollback); err != nil {
		return err
	}

	if breakpoint != "" {
		if err := m.runMigrator("", ""); err != nil {
			return err
		}
	}

	m.runCentral()

	return nil
}

func (m *mockCentral) restore(ver *versionPair, rocksToPostgres bool) {
	// Central should be in running state.
	m.verifyCurrent()

	if rocksToPostgres {
		restoreDir, err := os.MkdirTemp(migrations.DBMountPath(), ".restore-")
		require.NoError(m.t, os.Symlink(filepath.Base(restoreDir), filepath.Join(migrations.DBMountPath(), ".restore")))
		require.NoError(m.t, err)
		// backups from version lower than 3.0.57.0 do not have migration version.
		if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
			m.setMigrationVersion(restoreDir, ver)
		}
	}

	pgtest.CreateDatabase(m.t, migrations.RestoreDatabase)

	// backups from version lower than 3.0.57.0 do not have migration version.
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		m.setMigrationVersionPostgres(migrations.RestoreDatabase, ver)
	}
}

func (m *mockCentral) verifyCurrent() {
	m.verifyClonePostgres(pgClone.CurrentClone, &versionPair{version: version.GetMainVersion(), seqNum: migrations.CurrentDBVersionSeqNum(), minSeqNum: migrations.MinimumSupportedDBVersionSeqNum()})
}

func (m *mockCentral) verifyClone(clone string, ver *versionPair) {
	dbPath := filepath.Join(m.mountPath, clone)
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		m.verifyMigrationVersion(dbPath, ver)
	} else {
		require.NoFileExists(m.t, filepath.Join(dbPath, migrations.MigrationVersionFile))
		m.verifyMigrationVersion(dbPath, &versionPair{version: "0", seqNum: 0})
	}
}

func (m *mockCentral) verifyClonePostgres(clone string, ver *versionPair) {
	if version.CompareVersions(ver.version, "3.0.57.0") >= 0 {
		m.verifyMigrationVersionPostgres(clone, ver)
	} else {
		m.verifyMigrationVersionPostgres(clone, &versionPair{version: "0", seqNum: 0, minSeqNum: 0})
	}
}

func (m *mockCentral) setMigrationVersion(path string, ver *versionPair) {
	migVer := migrations.MigrationVersion{MainVersion: ver.version, SeqNum: ver.seqNum, MinimumSeqNum: ver.minSeqNum}
	bytes, err := yaml.Marshal(migVer)
	require.NoError(m.t, err)
	require.NoError(m.t, os.WriteFile(filepath.Join(path, migrations.MigrationVersionFile), bytes, 0644))
}

func (m *mockCentral) setMigrationVersionPostgres(clone string, ver *versionPair) {
	migVer.SetVersionPostgres(m.ctx, clone, &storage.Version{SeqNum: int32(ver.seqNum), Version: ver.version, MinSeqNum: int32(ver.minSeqNum)})
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
	assert.Equal(m.t, ver.minSeqNum, migVer.MinimumSeqNum)
	require.Equal(m.t, ver.version, migVer.MainVersion)
}

func (m *mockCentral) runMigratorWithBreaksInPersist(breakpoint string) {
	source := pgtest.GetConnectionString(m.t)
	sourceMap, _ := pgconfig.ParseSource(source)
	config, err := postgres.ParseConfig(source)
	require.NoError(m.t, err)

	dbm := pgClone.New("", config, sourceMap)
	err = dbm.Scan()
	require.NoError(m.t, err)
	clone, _, err := dbm.GetCloneToMigrate(nil)
	require.NoError(m.t, err)
	m.upgradeDB(clone)

	var prev string
	switch clone {
	case pgClone.CurrentClone:
		return
	case pgClone.PreviousClone:
		prev = ""
	case pgClone.RestoreClone:
		prev = pgClone.BackupClone
	case pgClone.TempClone:
		prev = pgClone.PreviousClone
	}

	// Connect to different database for admin functions
	connectPool, err := pgadmin.GetAdminPool(m.adminConfig)
	assert.NoError(m.t, err)
	// Close the admin connection pool
	defer connectPool.Close()

	log.Infof("runMigratorWithBreaksInPersist, prev = %s", prev)
	pgtest.DropDatabase(m.t, prev)
	if breakpoint == breakAfterRemove {
		return
	}
	_ = postgreshelper.RenameDB(connectPool, prev, pgClone.CurrentClone)
	if breakpoint == breakBeforeCommitCurrent {
		return
	}
	_ = postgreshelper.RenameDB(connectPool, pgClone.CurrentClone, clone)
}

func (m *mockCentral) removePreviousClone() {
	pgtest.DropDatabase(m.t, pgClone.PreviousClone)
}
