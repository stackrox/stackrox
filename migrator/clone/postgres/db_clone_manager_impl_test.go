//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/clone/metadata"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

var (
	preVer         = versionPair{version: "4.10.0", seqNum: 220, minSeqNum: 209}
	currVer        = versionPair{version: "5.1.0", seqNum: migrations.CurrentDBVersionSeqNum(), minSeqNum: 220}
	futureVer      = versionPair{version: "5.2.0", seqNum: 6533, minSeqNum: 2011}
	unsupportedVer = versionPair{version: "4.3.1", seqNum: 194, minSeqNum: 0} // Below minimum supported version (209 / 4.6)
)

type versionPair struct {
	version   string
	seqNum    int
	minSeqNum int
}

type PostgresCloneManagerSuite struct {
	suite.Suite
	pool      postgres.DB
	config    *postgres.Config
	sourceMap map[string]string
	ctx       context.Context
	gc        migGorm.Config
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(PostgresCloneManagerSuite))
}

func (s *PostgresCloneManagerSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
	s.Require().NoError(err)
	s.gc = migGorm.SetupAndGetMockConfig(s.T())

	s.ctx = ctx
	s.pool = pool
	s.config = config
	s.sourceMap, err = pgconfig.ParseSource(source)
	if err != nil {
		log.Infof("Unable to parse source %q", source)
	}

	s.setVersion(s.T(), &currVer)
	s.CreateClones()
}

func (s *PostgresCloneManagerSuite) CreateClones() {
	for clone := range knownClones {
		pgtest.CreateDatabase(s.T(), clone)
	}
}

func (s *PostgresCloneManagerSuite) DestroyClones() {
	// Clean up databases
	for clone := range knownClones {
		pgtest.DropDatabase(s.T(), clone)
	}
}

func (s *PostgresCloneManagerSuite) TearDownTest() {
	if s.pool != nil {
		s.DestroyClones()

		s.pool.Close()
	}
}

func (s *PostgresCloneManagerSuite) setVersion(t *testing.T, ver *versionPair) {
	log.Infof("setVersion => %v", ver)
	testutils.SetMainVersion(t, ver.version)
	migrationtestutils.SetCurrentDBSequenceNumber(t, ver.seqNum)
}

func (s *PostgresCloneManagerSuite) TestScan() {
	for clone := range knownClones {
		s.Require().True(pgadmin.CheckIfDBExists(s.config, clone))
	}

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	// Ensure known clones remain and temp clones are deleted
	for clone := range knownClones {
		s.Require().True(pgadmin.CheckIfDBExists(s.config, clone))
	}
}

func (s *PostgresCloneManagerSuite) TestScanRestoreFromFuture() {
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Set central_active in the future and have no previous
	futureVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() + 2),
		Version:       futureVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetRestoreClone(), futureVersion)

	s.Require().EqualError(dbm.Scan(), fmt.Sprintf(metadata.ErrUnableToRestore, futureVersion.GetVersion(), version.GetMainVersion()))
}

func (s *PostgresCloneManagerSuite) TestScanRestoreFromUnsupportedVersion() {
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Set central_restore to an unsupported old version (4.3.1, seq 194)
	// This is below the minimum supported version (4.6, seq 209)
	oldVersion := &storage.Version{
		SeqNum:        int32(unsupportedVer.seqNum),
		Version:       unsupportedVer.version,
		MinSeqNum:     int32(unsupportedVer.minSeqNum),
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetRestoreClone(), oldVersion)

	// The scan should detect that we're trying to restore from an unsupported version
	// and return an error indicating the restore is not compatible
	err := dbm.Scan()
	s.Require().Error(err, "Expected error when scanning restore from unsupported version")
	s.Require().Contains(err.Error(), "not supported", "Error message should indicate version not supported")
}

func (s *PostgresCloneManagerSuite) TestRestoreCompatibility() {
	// A valid restore must not be blocked by the new gate on the replaced database.
	migVer.SetVersionPostgres(s.ctx, CurrentClone, &storage.Version{Version: "4.9.0", SeqNum: 213})
	migVer.SetVersionPostgres(s.ctx, RestoreClone, &storage.Version{Version: "4.10.0", SeqNum: 220})
	dbm := New("", s.config, s.sourceMap)
	s.Require().NoError(dbm.Scan())
	selected, err := dbm.GetCloneToMigrate()
	s.Require().NoError(err)
	s.Equal(RestoreClone, selected)

	// Even an equal sequence must not make an N-4 restore compatible.
	migVer.SetVersionPostgres(s.ctx, RestoreClone, &storage.Version{Version: "4.9.0", SeqNum: 220})
	dbm = New("", s.config, s.sourceMap)
	s.Require().ErrorContains(dbm.Scan(), "4.9")
	ver, err := migVer.ReadVersionPostgres(s.ctx, RestoreClone)
	s.Require().NoError(err)
	s.Equal("4.9.0", ver.MainVersion)
	s.Equal(220, ver.SeqNum)

	// The unsafe product-version override must not weaken the restore sequence floor.
	s.T().Setenv("ROX_UNSAFE_ALLOW_UNSUPPORTED_UPGRADE", "true")
	migVer.SetVersionPostgres(s.ctx, RestoreClone, &storage.Version{Version: "4.9.0", SeqNum: 213})
	s.Require().ErrorContains(New("", s.config, s.sourceMap).Scan(), "not supported")
}

func (s *PostgresCloneManagerSuite) TestUpgradeCompatibility() {
	pgtest.DropDatabase(s.T(), RestoreClone)
	migVer.SetVersionPostgres(s.ctx, CurrentClone, &storage.Version{Version: "4.9.0", SeqNum: 227})
	s.Require().ErrorContains(New("", s.config, s.sourceMap).Scan(), "4.9")
	s.T().Setenv("ROX_UNSAFE_ALLOW_UNSUPPORTED_UPGRADE", "true")
	s.Require().NoError(New("", s.config, s.sourceMap).Scan())
}

func (s *PostgresCloneManagerSuite) TestGetRestoreClone() {
	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	clone, err := dbm.GetCloneToMigrate()
	s.Require().Nil(err)
	s.Require().Equal(clone, migrations.RestoreDatabase)
}

func (s *PostgresCloneManagerSuite) TestGetCloneFreshCurrent() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	clone, err := dbm.GetCloneToMigrate()
	s.Require().Equal(clone, CurrentClone)
	s.Require().Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneCurrentCurrent() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	clone, err := dbm.GetCloneToMigrate()
	s.Require().Equal(CurrentClone, clone)
	s.Require().Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneUpgrade() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() - 2),
		Version:       preVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	clone, err := dbm.GetCloneToMigrate()
	s.Require().Equal(migrations.CurrentDatabase, clone)
	s.Require().Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneUpgradeSameSeq() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       preVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	clone, err := dbm.GetCloneToMigrate()
	s.Require().Equal(CurrentClone, clone)
	s.Require().Nil(err)
}

func (s *PostgresCloneManagerSuite) TestPersistCurrentClone() {
	// Test persisting CurrentClone - should be a no-op
	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	// Verify CurrentClone exists before persist
	exists, err := pgadmin.CheckIfDBExists(s.config, CurrentClone)
	s.Require().NoError(err)
	s.Require().True(exists)

	// Persist CurrentClone - should not error and should be a no-op
	err = dbm.Persist(CurrentClone)
	s.Require().NoError(err)

	// Verify CurrentClone still exists after persist
	exists, err = pgadmin.CheckIfDBExists(s.config, CurrentClone)
	s.Require().NoError(err)
	s.Require().True(exists)
}

func (s *PostgresCloneManagerSuite) TestPersistRestoreClone() {
	// Drop backup database to ensure clean state
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Create and populate restore clone with a version
	restoreVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() - 1),
		Version:       preVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetRestoreClone(), restoreVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	// Verify restore clone exists
	exists, err := pgadmin.CheckIfDBExists(s.config, RestoreClone)
	s.Require().NoError(err)
	s.Require().True(exists)

	// Verify current clone exists
	exists, err = pgadmin.CheckIfDBExists(s.config, CurrentClone)
	s.Require().NoError(err)
	s.Require().True(exists)

	// Persist restore clone
	err = dbm.Persist(RestoreClone)
	s.Require().NoError(err)

	// After persist:
	// - RestoreClone should no longer exist (it was renamed to CurrentClone)
	// - CurrentClone should exist (the former RestoreClone)
	// - BackupClone should exist (the former CurrentClone)
	exists, err = pgadmin.CheckIfDBExists(s.config, RestoreClone)
	s.Require().NoError(err)
	s.Require().False(exists, "RestoreClone should not exist after persist")

	exists, err = pgadmin.CheckIfDBExists(s.config, CurrentClone)
	s.Require().NoError(err)
	s.Require().True(exists, "CurrentClone should exist after persist")

	exists, err = pgadmin.CheckIfDBExists(s.config, BackupClone)
	s.Require().NoError(err)
	s.Require().True(exists, "BackupClone should exist after persist")
}

func (s *PostgresCloneManagerSuite) TestPersistRestoreCloneWithExistingBackup() {
	// Create a backup database first
	backupVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() - 2),
		Version:       preVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetBackupClone(), backupVersion)

	// Create and populate restore clone with a version
	restoreVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() - 1),
		Version:       preVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetRestoreClone(), restoreVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	// Verify all clones exist before persist
	exists, err := pgadmin.CheckIfDBExists(s.config, RestoreClone)
	s.Require().NoError(err)
	s.Require().True(exists)

	exists, err = pgadmin.CheckIfDBExists(s.config, CurrentClone)
	s.Require().NoError(err)
	s.Require().True(exists)

	exists, err = pgadmin.CheckIfDBExists(s.config, BackupClone)
	s.Require().NoError(err)
	s.Require().True(exists)

	// Persist restore clone (should remove existing backup first)
	err = dbm.Persist(RestoreClone)
	s.Require().NoError(err)

	// After persist:
	// - RestoreClone should not exist (renamed to CurrentClone)
	// - CurrentClone should exist (the former RestoreClone)
	// - BackupClone should exist (the former CurrentClone, old backup was removed)
	exists, err = pgadmin.CheckIfDBExists(s.config, RestoreClone)
	s.Require().NoError(err)
	s.Require().False(exists, "RestoreClone should not exist after persist")

	exists, err = pgadmin.CheckIfDBExists(s.config, CurrentClone)
	s.Require().NoError(err)
	s.Require().True(exists, "CurrentClone should exist after persist")

	exists, err = pgadmin.CheckIfDBExists(s.config, BackupClone)
	s.Require().NoError(err)
	s.Require().True(exists, "BackupClone should exist after persist")
}

func (s *PostgresCloneManagerSuite) TestPersistRestoreCloneWithoutCurrentClone() {
	// Drop current clone to simulate fresh startup scenario
	pgtest.DropDatabase(s.T(), migrations.CurrentDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Create restore clone
	restoreVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetRestoreClone(), restoreVersion)

	// Recreate current clone for the test setup
	pgtest.CreateDatabase(s.T(), migrations.CurrentDatabase)
	currentVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currentVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Require().Nil(dbm.Scan())

	// Persist restore clone
	err := dbm.Persist(RestoreClone)
	s.Require().NoError(err)

	// Verify the outcome
	exists, err := pgadmin.CheckIfDBExists(s.config, RestoreClone)
	s.Require().NoError(err)
	s.Require().False(exists, "RestoreClone should not exist after persist")

	exists, err = pgadmin.CheckIfDBExists(s.config, CurrentClone)
	s.Require().NoError(err)
	s.Require().True(exists, "CurrentClone should exist after persist")
}
