package restore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type PostgresRestoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	pool        *pgxpool.Pool
	config      *pgxpool.Config
	sourceMap   map[string]string
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

	s.pool = pool
	s.config = config
	s.sourceMap, err = globaldb.ParseSource(source)
	if err != nil {
		log.Infof("Unable to parse source %q", source)
	}
}

func (s *PostgresRestoreSuite) TearDownTest() {

	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}

func (s *PostgresRestoreSuite) TestRestoreUtilities() {
	// Drop the restore DB if it is lingering from a previous test.
	// Clean up any databases that were created
	_ = dropDB(s.sourceMap, s.config, restoreDB)

	// Everything fresh.  A restor database should not exist.
	s.False(CheckIfRestoreDBExists(s.config))

	// Create a restore DB
	err := createDB(s.sourceMap, s.config)
	s.Nil(err)

	// Verify restore DB was created
	s.True(CheckIfRestoreDBExists(s.config))

	// Make a copy of the config and set a user that does not have
	// permissions to create, drop, etc
	badConfig := s.config.Copy()
	badConfig.ConnConfig.User = "baduser"

	// Fail to create restore DB because of insufficient user permissions
	err = createDB(s.sourceMap, badConfig)
	s.NotNil(err)

	// Fail to drop restore DB because of insufficient user permissions
	err = dropDB(s.sourceMap, badConfig, restoreDB)
	s.NotNil(err)

	// Successfully drop the restore DB
	err = dropDB(s.sourceMap, s.config, restoreDB)
	s.Nil(err)
}
