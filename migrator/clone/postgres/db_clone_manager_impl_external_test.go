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
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
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

	clone, err := dbm.GetCloneToMigrate()
	s.Equal(externalDB, clone)
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

	s.True(pgadmin.CheckIfDBExists(s.config, externalDB))
}

func (s *PostgresExternalManagerSuite) TestScanIncompatibleExternal() {
	dbm := New(currVer.version, s.config, s.sourceMap)

	// Set central_active in the future and have no previous
	futureVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum() + 2),
		Version:       futureVer.version,
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
		MinSeqNum:     int32(migrations.CurrentDBVersionSeqNum() + 2),
	}
	migVer.SetVersionPostgres(s.ctx, externalDB, futureVersion)

	// Scan the clones
	errorMessage := fmt.Sprintf(metadata.ErrSoftwareNotCompatibleWithDatabase, migrations.CurrentDBVersionSeqNum(), futureVersion.GetMinSeqNum(), migrations.MinimumSupportedDBVersion())
	s.EqualError(dbm.Scan(), errorMessage)
}
