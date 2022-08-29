package pgadmin

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const (
	activeDB  = migrations.CurrentDatabase
	restoreDB = migrations.RestoreDatabase
	tempDB    = "central_temp"

	// Database with no typical connections that will be used as a template in a create
	adminDB = "template1"
)

type PostgresRestoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	pool        *pgxpool.Pool
	config      *pgxpool.Config
	sourceMap   map[string]string
	ctx         context.Context
}

func TestRestore(t *testing.T) {
	suite.Run(t, new(PostgresRestoreSuite))
}

func (s *PostgresRestoreSuite) SetupTest() {
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
}

func (s *PostgresRestoreSuite) TearDownTest() {
	//Clean up
	err := DropDB(s.sourceMap, s.config, restoreDB)
	s.Nil(err)
	err = DropDB(s.sourceMap, s.config, activeDB)
	s.Nil(err)

	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}

func (s *PostgresRestoreSuite) TestUtilities() {
	// Drop the restore DB if it is lingering from a previous test.
	// Clean up any databases that were created
	_ = DropDB(s.sourceMap, s.config, restoreDB)

	// Everything fresh.  A restore database should not exist.
	s.False(CheckIfDBExists(s.config, restoreDB))

	// Create a restore DB
	err := CreateDB(s.sourceMap, s.config, adminDB, restoreDB)
	s.Nil(err)

	// Verify restore DB was created
	s.True(CheckIfDBExists(s.config, restoreDB))

	// Get a connection to the restore database
	restorePool := GetClonePool(s.config, restoreDB)
	s.NotNil(restorePool)
	err = restorePool.Ping(s.ctx)
	s.Nil(err)

	// Successfully create active DB from restore DB
	err = CreateDB(s.sourceMap, s.config, restoreDB, activeDB)
	s.Nil(err)
	// Have to terminate connections from the source DB before we can create
	// the copy.  Make sure connection was terminated.
	err = restorePool.Ping(s.ctx)
	s.NotNil(err)

	// Rename database to a database that exists
	err = RenameDB(s.pool, restoreDB, activeDB)
	s.NotNil(err)

	// Get a connection to the active DB
	activePool := GetClonePool(s.config, activeDB)
	s.NotNil(activePool)

	// Rename activeDB to a new one
	err = RenameDB(s.pool, activeDB, tempDB)
	s.Nil(err)
	s.True(CheckIfDBExists(s.config, tempDB))
	// Make sure connection to active database was terminated
	s.NotNil(activePool.Ping(s.ctx))

	// Reacquire a connection to the restore database
	restorePool = GetClonePool(s.config, restoreDB)
	s.NotNil(restorePool)
	s.Nil(restorePool.Ping(s.ctx))

	// Successfully drop the restore DB
	err = DropDB(s.sourceMap, s.config, restoreDB)
	s.Nil(err)
	s.NotNil(restorePool.Ping(s.ctx))
}
