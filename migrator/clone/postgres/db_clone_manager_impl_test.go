package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/clone/metadata"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

const (
	tempDB = "central_temp"

	// Database with no typical connections that will be used as a template in a create
	adminDB = "template1"
)

var (
	preHistoryVer = versionPair{version: "3.0.56.0", seqNum: 62}
	preVer        = versionPair{version: "3.0.57.0", seqNum: 65}
	currVer       = versionPair{version: "3.0.58.0", seqNum: 65}
	futureVer     = versionPair{version: "10001.0.0.0", seqNum: 6533}

	// Current versions
	rcVer      = versionPair{version: "3.0.58.0-rc.1", seqNum: 65}
	releaseVer = versionPair{version: "3.0.58.0", seqNum: 65}
	devVer     = versionPair{version: "3.0.58.x-19-g6bd31dae22-dirty", seqNum: 65}
	nightlyVer = versionPair{version: "3.0.58.x-nightly-20210407", seqNum: 65}

	invalidClones = set.NewStringSet(tempDB, "central_123", "central_garbage")
)

type versionPair struct {
	version string
	seqNum  int
}

type PostgresCloneManagerSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	pool        *pgxpool.Pool
	config      *pgxpool.Config
	sourceMap   map[string]string
	ctx         context.Context
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(PostgresCloneManagerSuite))
}

func (s *PostgresCloneManagerSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.Require().NoError(err)

	s.ctx = ctx
	s.pool = pool
	s.config = config
	s.sourceMap, err = pgconfig.ParseSource(source)
	if err != nil {
		log.Infof("Unable to parse source %q", source)
	}

	s.setVersion(s.T(), &currVer)
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

	s.envIsolator.RestoreAll()
}

func (s *PostgresCloneManagerSuite) setVersion(t *testing.T, ver *versionPair) {
	log.Infof("setVersion => %v", ver)
	testutils.SetMainVersion(t, ver.version)
	migrationtestutils.SetCurrentDBSequenceNumber(t, ver.seqNum)
}

func (s *PostgresCloneManagerSuite) TestScan() {
	s.CreateClones()

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

	// Clean up for the next test
	s.DestroyClones()
}

func (s *PostgresCloneManagerSuite) TestScanCurrentPrevious() {
	s.CreateClones()
	pgtest.DropDatabase(s.T(), tempDB)
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Set central_active in the future and have no previous
	pool := pgadmin.GetClonePool(s.config, migrations.GetCurrentClone())
	futureVersion := &storage.Version{
		SeqNum:        int32(futureVer.seqNum),
		Version:       futureVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, futureVersion)
	pool.Close()

	// Drop previous
	pgtest.DropDatabase(s.T(), migrations.PreviousDatabase)

	// Scan the clones
	s.EqualError(dbm.Scan(), metadata.ErrNoPrevious)

	// Create a previous and set its version to current one
	pgtest.CreateDatabase(s.T(), migrations.PreviousDatabase)
	pool = pgadmin.GetClonePool(s.config, migrations.PreviousDatabase)
	verForPrevClone := &storage.Version{
		SeqNum:        int32(currVer.seqNum),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, verForPrevClone)
	pool.Close()

	// Scan the clones
	s.EqualError(dbm.Scan(), metadata.ErrForceUpgradeDisabled)

	// Set previous clone version so it doesn't match current sw version
	pool = pgadmin.GetClonePool(s.config, migrations.PreviousDatabase)
	verForPrevClone = &storage.Version{
		SeqNum:        int32(currVer.seqNum),
		Version:       preVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, verForPrevClone)
	pool.Close()

	// New manager with force rollback version set
	dbm = New(currVer.version, s.config, s.sourceMap)
	s.EqualError(dbm.Scan(), fmt.Sprintf(metadata.ErrPreviousMismatchWithVersions, verForPrevClone.GetVersion(), version.GetMainVersion()))

	// Clean up for the next test
	s.DestroyClones()
}

func (s *PostgresCloneManagerSuite) TestScanRestoreFromFuture() {
	s.CreateClones()
	pgtest.DropDatabase(s.T(), tempDB)
	pgtest.DropDatabase(s.T(), migrations.PreviousDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Set central_active in the future and have no previous
	pool := pgadmin.GetClonePool(s.config, migrations.RestoreDatabase)
	futureVersion := &storage.Version{
		SeqNum:        int32(futureVer.seqNum),
		Version:       futureVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, futureVersion)
	pool.Close()

	s.EqualError(dbm.Scan(), fmt.Sprintf(metadata.ErrUnableToRestore, futureVersion.GetVersion(), version.GetMainVersion()))

	// Clean up for the next test
	s.DestroyClones()
}

func (s *PostgresCloneManagerSuite) TestGetRestoreClone() {
	s.CreateClones()

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil)
	s.Equal(clone, migrations.RestoreDatabase)
	s.False(migrateRocks)
	s.Nil(err)

	// Clean up for the next test
	s.DestroyClones()
}

func (s *PostgresCloneManagerSuite) TestGetCloneMigrateRocks() {
	s.CreateClones()
	pgtest.DropDatabase(s.T(), tempDB)
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	pool := pgadmin.GetClonePool(s.config, migrations.GetCurrentClone())
	currVersion := &storage.Version{
		SeqNum:        int32(currVer.seqNum),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, currVersion)
	pool.Close()

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	rocksVersion := &migrations.MigrationVersion{
		SeqNum:        currVer.seqNum,
		MainVersion:   currVer.version,
		LastPersisted: time.Now(),
	}

	// Need to migrate from Rocks because Rocks is more current.
	clone, migrateRocks, err := dbm.GetCloneToMigrate(rocksVersion)
	s.Equal(clone, tempDB)
	s.True(migrateRocks)
	s.Nil(err)

	rocksVersion = &migrations.MigrationVersion{
		SeqNum:        currVer.seqNum,
		MainVersion:   currVer.version,
		LastPersisted: time.Now().Add(-time.Hour * 24),
	}

	// Need to migrate from Rocks because Rocks is more current.
	clone, migrateRocks, err = dbm.GetCloneToMigrate(rocksVersion)
	s.Equal(clone, CurrentClone)
	s.False(migrateRocks)
	s.Nil(err)

	// Need to migrate from Rocks because it is newer.
	rocksVersion = &migrations.MigrationVersion{
		SeqNum:        currVer.seqNum,
		MainVersion:   currVer.version,
		LastPersisted: time.Now().Add(time.Hour * 24),
	}
	// Need to re-scan to get the clone deletion
	s.Nil(dbm.Scan())
	clone, migrateRocks, err = dbm.GetCloneToMigrate(rocksVersion)
	s.Equal(clone, tempDB)
	s.True(migrateRocks)
	s.Nil(err)

	// Clean up for the next test
	s.DestroyClones()
}

func (s *PostgresCloneManagerSuite) TestGetCloneFreshCurrent() {
	s.CreateClones()
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil)
	s.Equal(clone, CurrentClone)
	s.False(migrateRocks)
	s.Nil(err)

	// Clean up for the next test
	s.DestroyClones()
}

func (s *PostgresCloneManagerSuite) TestGetCloneCurrentCurrent() {
	s.CreateClones()
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	pool := pgadmin.GetClonePool(s.config, migrations.GetCurrentClone())
	currVersion := &storage.Version{
		SeqNum:        int32(currVer.seqNum),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, currVersion)
	pool.Close()

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil)
	s.Equal(clone, CurrentClone)
	s.False(migrateRocks)
	s.Nil(err)

	// Clean up for the next test
	s.DestroyClones()
}

func (s *PostgresCloneManagerSuite) TestGetClonePrevious() {
	s.CreateClones()
	pgtest.DropDatabase(s.T(), migrations.RestoreDatabase)
	pgtest.DropDatabase(s.T(), migrations.BackupDatabase)

	// Set central_active in the future and have no previous
	pool := pgadmin.GetClonePool(s.config, migrations.GetCurrentClone())
	futureVersion := &storage.Version{
		SeqNum:        int32(futureVer.seqNum),
		Version:       futureVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, futureVersion)
	pool.Close()

	// Set previous to the current version to simulate a rollback
	pool = pgadmin.GetClonePool(s.config, migrations.PreviousDatabase)
	currVersion := &storage.Version{
		SeqNum:        int32(currVer.seqNum),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migrations.SetVersionPostgres(pool, currVersion)
	pool.Close()

	dbm := New(currVer.version, s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil)
	s.Equal(clone, PreviousClone)
	s.False(migrateRocks)
	s.Nil(err)

	// Clean up for the next test
	s.DestroyClones()
}
