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
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

const (
	externalDB = "stackrox"
)

type PostgresExternalManagerSuite struct {
	suite.Suite
	pool      postgres.DB
	config    *postgres.Config
	sourceMap map[string]string
	ctx       context.Context
	gc        migGorm.Config
}

func TestExternalManagerSuite(t *testing.T) {
	suite.Run(t, new(PostgresExternalManagerSuite))
}

func (s *PostgresExternalManagerSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())

	// Create the external database.
	pgtest.DropDatabase(s.T(), externalDB)
	pgtest.CreateDatabase(s.T(), externalDB)

	// Update the default configs to use the external database
	config.GetConfig().CentralDB.External = true

	source := pgtest.GetConnectionStringWithDatabaseName(s.T(), externalDB)
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
	s.Require().NoError(err)
	s.gc = migGorm.SetupAndGetMockConfigWithDatabase(s.T(), externalDB)

	s.ctx = ctx
	s.pool = pool
	s.config = config
	s.sourceMap, err = pgconfig.ParseSource(source)
	s.Require().NoError(err)

	s.setVersion(s.T(), &currVer)
}

func (s *PostgresExternalManagerSuite) DestroyClones() {
	// Clean up databases
	pgtest.DropDatabase(s.T(), tempDB)
	pgtest.DropDatabase(s.T(), externalDB)

	for clone := range knownClones {
		pgtest.DropDatabase(s.T(), clone)
	}
}

func (s *PostgresExternalManagerSuite) TearDownTest() {
	if s.pool != nil {
		s.DestroyClones()

		s.pool.Close()
	}

	// reset the external flag
	config.GetConfig().CentralDB.External = false
}

func (s *PostgresExternalManagerSuite) setVersion(t *testing.T, ver *versionPair) {
	log.Infof("setVersion => %v", ver)
	testutils.SetMainVersion(t, ver.version)
	migrationtestutils.SetCurrentDBSequenceNumber(t, ver.seqNum)
}

func (s *PostgresExternalManagerSuite) TestGetCloneFreshExternal() {
	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, false)
	s.Equal(externalDB, clone)
	s.False(migrateRocks)
	s.Nil(err)
}

func (s *PostgresExternalManagerSuite) TestGetRestoreFromRocksExternal() {
	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	clone, migrateRocks, err := dbm.GetCloneToMigrate(nil, true)
	s.Equal(clone, externalDB)
	s.True(migrateRocks)
	s.Nil(err)
}

func (s *PostgresExternalManagerSuite) TestScanExternal() {
	for clone := range knownClones {
		s.False(pgadmin.CheckIfDBExists(s.config, clone))
	}

	dbm := New("", s.config, s.sourceMap)

	// Scan the clones
	s.Nil(dbm.Scan())

	// Ensure known clones remain and temp clones are deleted
	for clone := range knownClones {
		s.False(pgadmin.CheckIfDBExists(s.config, clone))
	}

	s.False(pgadmin.CheckIfDBExists(s.config, tempDB))
	s.True(pgadmin.CheckIfDBExists(s.config, externalDB))
}

func (s *PostgresExternalManagerSuite) TestScanIncompatibleExternal() {
	dbm := New(currVer.version, s.config, s.sourceMap)

	// Set central_active in the future and have no previous
	futureVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() + 2),
		Version:       futureVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
		MinSeqNum:     int32(migrations.MinimumSupportedDBVersionSeqNum() + 2),
	}
	migVer.SetVersionPostgres(s.ctx, externalDB, futureVersion)

	// Drop previous
	pgtest.DropDatabase(s.T(), migrations.PreviousDatabase)

	// Scan the clones
	errorMessage := fmt.Sprintf(metadata.ErrSoftwareNotCompatibleWithDatabase, migrations.MinimumSupportedDBVersionSeqNum(), futureVersion.MinSeqNum)
	s.EqualError(dbm.Scan(), errorMessage)
}

func (s *PostgresExternalManagerSuite) TestExternalMigrateRocks() {
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
	s.Equal(clone, externalDB)
	s.True(migrateRocks)
	s.Nil(err)

	// Set central_active version to be in the middle of Rocks -> Postgres migrations
	currVersion := &storage.Version{
		SeqNum:        122,
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, externalDB, currVersion)

	// Need to re-scan to get the updated clone version
	s.Nil(dbm.Scan())
	// Need to use the Postgres database so migrateRocks will be false.
	clone, migrateRocks, err = dbm.GetCloneToMigrate(rocksVersion, false)
	s.Equal(clone, externalDB)
	s.True(migrateRocks)
	s.Nil(err)

	// Set central_active version
	currVersion = &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       currVer.version,
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	migVer.SetVersionPostgres(s.ctx, externalDB, currVersion)

	// Need to re-scan to get the updated clone version
	s.Nil(dbm.Scan())
	// Need to use the Postgres database so migrateRocks will be false.
	clone, migrateRocks, err = dbm.GetCloneToMigrate(rocksVersion, false)
	s.Equal(clone, externalDB)
	s.False(migrateRocks)
	s.Nil(err)
}
