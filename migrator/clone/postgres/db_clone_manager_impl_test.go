//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

const (
	tempDB = TempClone
)

var (
	preVer    = versionPair{version: "3.0.57.0", seqNum: 63, minSeqNum: 0}
	currVer   = versionPair{version: "3.0.58.0", seqNum: migrations.CurrentDBVersionSeqNum(), minSeqNum: migrations.MinimumSupportedDBVersionSeqNum()}
	futureVer = versionPair{version: "10001.0.0.0", seqNum: 6533, minSeqNum: 2011}
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
	pgtest.CreateDatabase(s.T(), tempDB)

	for clone := range knownClones {
		pgtest.CreateDatabase(s.T(), clone)
	}
}

func (s *PostgresCloneManagerSuite) DestroyClones() {
	// Clean up databases
	pgtest.DropDatabase(s.T(), tempDB)

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
	s.True(pgadmin.CheckIfDBExists(s.config, tempDB))

	for clone := range knownClones {
		s.True(pgadmin.CheckIfDBExists(s.config, clone))
	}

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	// Ensure known clones remain and temp clones are deleted
	for clone := range knownClones {
		s.True(pgadmin.CheckIfDBExists(s.config, clone))
	}

	s.False(pgadmin.CheckIfDBExists(s.config, tempDB))
}

func (s *PostgresCloneManagerSuite) TestScanCurrentPrevious() {
	pgtest.DropDatabase(s.T(), tempDB)
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New(currVer.version, s.config, s.sourceMap)

	// Set central_active in the future and have no previous
	futureVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() + 2),
		Version:       futureVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
		MinSeqNum:     int32(migrations.MinimumSupportedDBVersionSeqNum() + 2),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), futureVersion)

	// Drop previous
	pgtest.DropDatabase(s.T(), migrations.PreviousDatabase)

	// Scan the clones
	errorMessage := fmt.Sprintf(metadata.ErrSoftwareNotCompatibleWithDatabase, migrations.MinimumSupportedDBVersionSeqNum(), futureVersion.MinSeqNum)
	s.EqualError(dbm.Scan(), errorMessage)

	// Create a previous and set its version to current one
	pgtest.CreateDatabase(s.T(), migrations.PreviousDatabase)
	verForPrevClone := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetPreviousClone(), verForPrevClone)

	// Scan the clones
	s.EqualError(dbm.Scan(), errorMessage)

	// Set previous clone version so it doesn't match current sw version
	verForPrevClone = &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() - 2),
		Version:       preVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetPreviousClone(), verForPrevClone)

	// New manager with force rollback version set
	dbm = New(currVer.version, s.config, s.sourceMap)
	s.EqualError(dbm.Scan(), errorMessage)
}

func (s *PostgresCloneManagerSuite) TestScanRestoreFromFuture() {
	pgtest.DropDatabase(s.T(), tempDB)
	pgtest.DropDatabase(s.T(), migrations.PreviousDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Set central_active in the future and have no previous
	futureVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() + 2),
		Version:       futureVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetRestoreClone(), futureVersion)

	s.EqualError(dbm.Scan(), fmt.Sprintf(metadata.ErrUnableToRestore, futureVersion.GetVersion(), version.GetMainVersion()))
}

func (s *PostgresCloneManagerSuite) TestGetRestoreClone() {
	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, false)
	s.Equal(clone, migrations.RestoreDatabase)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneMigrateRocks() {
	pgtest.DropDatabase(s.T(), tempDB)
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	rocksVersion := &migrations.MigrationVersion{
		SeqNum:        currVer.seqNum,
		MainVersion:   currVer.version,
		LastPersisted: time.Now(),
	}

	// No central_active exists so we return the temp clone to use and migrate to rocks
	clone, migrateRocks, err := dbm.GetCloneToMigrate(rocksVersion, false)
	s.Equal(CurrentClone, clone)
	s.True(migrateRocks)
	s.Nil(err)

	// Need to migrate from Rocks because Rocks exists and Postgres is fresh.
	pgtest.CreateDatabase(s.T(), migrations.CurrentDatabase)

	// Still migrate from Rocks because no version in Postgres meaning it is empty
	clone, migrateRocks, err = dbm.GetCloneToMigrate(rocksVersion, false)
	s.Equal(CurrentClone, clone)
	s.True(migrateRocks)
	s.Nil(err)

	// Set central_active version
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currVersion)

	// Need to re-scan to get the updated clone version
	s.Nil(dbm.Scan())
	// Need to use use the Postgres database so migrateRocks will be false.
	clone, migrateRocks, err = dbm.GetCloneToMigrate(rocksVersion, false)
	s.Equal(clone, CurrentClone)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneFreshCurrent() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, false)
	s.Equal(clone, CurrentClone)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneCurrentCurrent() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, false)
	s.Equal(CurrentClone, clone)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneUpgrade() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() - 2),
		Version:       preVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, false)
	s.Equal(TempClone, clone)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetCloneUpgradeSameSeq() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       preVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), currVersion)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, false)
	s.Equal(CurrentClone, clone)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetClonePrevious() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	futureVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() + 2),
		Version:       futureVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
		MinSeqNum:     int32(migrations.MinimumSupportedDBVersionSeqNum()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetCurrentClone(), futureVersion)

	// Set previous to the current version to simulate a rollback
	currVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
		MinSeqNum:     int32(migrations.MinimumSupportedDBVersionSeqNum()),
	}
	migVer.SetVersionPostgres(s.ctx, migrations.GetPreviousClone(), currVersion)

	dbm := New(currVer.version, s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, false)
	s.Equal(clone, CurrentClone)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresCloneManagerSuite) TestGetRestoreFromRocksClone() {
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, true)
	s.Equal(clone, migrations.RestoreDatabase)
	s.True(migrateRocks)
	s.Nil(err)
}
