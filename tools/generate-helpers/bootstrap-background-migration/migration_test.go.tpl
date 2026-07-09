{{- define "TODO"}}TODO(do{{- /**/ -}}nt-merge){{end -}}
//go:build sql_integration

package {{.packageName}}

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pgtest.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.db = pgtest.ForT(s.T())
	// {{template "TODO"}}: Create the table(s) required for the pre-migration dataset
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	// {{template "TODO"}}: Insert pre-migration test data

	// Run the migration
	s.Require().NoError(run(s.ctx, s.db.DB))

	// {{template "TODO"}}: Verify post-migration state (correct values backfilled)

	// Verify idempotency: running again should be a no-op
	s.Require().NoError(run(s.ctx, s.db.DB))

	// {{template "TODO"}}: Verify state is unchanged after second run
}

func (s *migrationTestSuite) TestGracefulShutdown() {
	// {{template "TODO"}}: Insert test data, cancel context mid-migration,
	// verify no data corruption, then complete with fresh context
}
