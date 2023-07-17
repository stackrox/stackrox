//go:build sql_integration

package pgadmin

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
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
	pool      postgres.DB
	config    *postgres.Config
	sourceMap map[string]string
	ctx       context.Context
}

func TestRestore(t *testing.T) {
	suite.Run(t, new(PostgresRestoreSuite))
}

func (s *PostgresRestoreSuite) SetupTest() {

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
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
	if s.pool != nil {
		// Clean up
		s.Nil(DropDB(s.sourceMap, s.config, restoreDB))
		s.Nil(DropDB(s.sourceMap, s.config, activeDB))
		s.Nil(DropDB(s.sourceMap, s.config, tempDB))
		s.pool.Close()
	}
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
	restorePool, err := GetClonePool(s.config, restoreDB)
	s.Nil(err)
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

	// Reacquire a connection to the restore database
	restorePool, err = GetClonePool(s.config, restoreDB)
	s.Nil(err)
	s.NotNil(restorePool)
	s.Nil(restorePool.Ping(s.ctx))

	// Successfully drop the restore DB
	err = DropDB(s.sourceMap, s.config, restoreDB)
	s.Nil(err)
	// Make sure the connection to the restore db was terminated
	s.NotNil(restorePool.Ping(s.ctx))
}
