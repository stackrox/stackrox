//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestDBDiagnostic(t *testing.T) {
	suite.Run(t, new(DBDiagnosticTestSuite))
}

type DBDiagnosticTestSuite struct {
	suite.Suite

	ctx      context.Context
	dbConfig *postgres.Config
	dbPool   postgres.DB
}

func (s *DBDiagnosticTestSuite) SetupSuite() {
	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
	s.Require().NoError(err)

	s.ctx = ctx
	s.dbPool = pool
	s.dbConfig = config
}

func (s *DBDiagnosticTestSuite) TearSuite() {
	if s.dbPool != nil {
		// Clean up
		s.dbPool.Close()
	}
}

func (s *DBDiagnosticTestSuite) TestClientVersion() {
	version := getDBClientVersion()
	s.NotEmpty(version)
}

func (s *DBDiagnosticTestSuite) TestExtensions() {
	extensions := getPostgresExtensions(s.ctx, s.dbPool)
	s.True(len(extensions) > 0)
}

func (s *DBDiagnosticTestSuite) TestBuildDiagnosticData() {
	diagnosticData := buildDBDiagnosticData(s.ctx, s.dbConfig, s.dbPool)
	s.True(len(diagnosticData.DatabaseExtensions) > 0)
	s.NotEmpty(diagnosticData.Database)
	s.NotEmpty(diagnosticData.DatabaseClientVersion)
	s.NotEmpty(diagnosticData.DatabaseServerVersion)
	s.NotEmpty(diagnosticData.DatabaseConnectString)
	s.Contains(diagnosticData.DatabaseConnectString, "REDACTED")

	// Drive some error cases
	diagnosticData = buildDBDiagnosticData(s.ctx, nil, s.dbPool)
	s.Equal(diagnosticData.DatabaseConnectString, "")
}
