//go:build sql_integration

package m227tom228

import (
	"context"
	"strings"
	"testing"

	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_227_to_m_228_drop_duplicate_policy_index/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), frozenSchema.CreateTablePoliciesStmt)
	// Concurrent indexes are applied separately from GORM table creation.
	for _, index := range frozenSchema.CreateTablePoliciesStmt.Indexes {
		_, err := s.db.Exec(s.ctx, index.CreateSQL)
		s.Require().NoError(err)
	}

	// Use enough rows and current statistics for selective lookups to naturally
	// choose an index; do not force index scans with enable_seqscan = off.
	_, err := s.db.Exec(s.ctx, `INSERT INTO policies (id, name, serialized)
		SELECT md5(n::text), 'policy-' || n, convert_to('policy-' || n, 'UTF8')
		FROM generate_series(1, 10000) AS n`)
	s.Require().NoError(err)
	_, err = s.db.Exec(s.ctx, "ANALYZE policies")
	s.Require().NoError(err)
}

func (s *migrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) dbs() *types.Databases {
	return &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}
}

func (s *migrationTestSuite) indexExists(name string) bool {
	var exists bool
	s.Require().NoError(s.db.QueryRow(s.ctx, `SELECT EXISTS (
		SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1
	)`, name).Scan(&exists))
	return exists
}

func (s *migrationTestSuite) TestMigration() {
	s.Require().True(s.indexExists("policies_id"))
	s.Require().True(s.indexExists("policies_pkey"))
	s.Require().True(s.indexExists("uni_policies_name"))

	for attempt := 1; attempt <= 2; attempt++ {
		s.Require().NoError(migration.Run(s.dbs()), "migration attempt %d", attempt)
		s.Require().False(s.indexExists("policies_id"))
		s.Require().True(s.indexExists("policies_pkey"))
		s.Require().True(s.indexExists("uni_policies_name"))

		// Both runs must leave every policy and its stored contents unchanged.
		var count int
		s.Require().NoError(s.db.QueryRow(s.ctx, `SELECT count(*) FROM policies
			WHERE name ~ '^policy-[0-9]+$'
			AND id = md5(substring(name FROM 8))
			AND serialized = convert_to(name, 'UTF8')`).Scan(&count))
		s.Equal(10000, count)
		s.Require().NoError(s.db.QueryRow(s.ctx, "SELECT count(*) FROM policies").Scan(&count))
		s.Equal(10000, count)
	}
}

func (s *migrationTestSuite) TestIDQueriesUsePrimaryKey() {
	s.Require().NoError(migration.Run(s.dbs()))

	const firstID = "c4ca4238a0b923820dcc509a6f75849b"
	const secondID = "c81e728d9d4c2f636f067f89cc14862c"
	testCases := map[string]struct {
		query    string
		args     []any
		expected map[string]string
	}{
		"single ID": {
			query:    "SELECT id, serialized FROM policies WHERE id = $1",
			args:     []any{firstID},
			expected: map[string]string{firstID: "policy-1"},
		},
		"multiple IDs": {
			query:    "SELECT id, serialized FROM policies WHERE id IN ($1, $2)",
			args:     []any{firstID, secondID},
			expected: map[string]string{firstID: "policy-1", secondID: "policy-2"},
		},
		"ID array": {
			query:    "SELECT id, serialized FROM policies WHERE id = ANY($1::text[])",
			args:     []any{[]string{firstID, secondID}},
			expected: map[string]string{firstID: "policy-1", secondID: "policy-2"},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			rows, err := s.db.Query(s.ctx, tc.query, tc.args...)
			s.Require().NoError(err)
			defer rows.Close()

			actual := make(map[string]string)
			for rows.Next() {
				var id string
				var serialized []byte
				s.Require().NoError(rows.Scan(&id, &serialized))
				actual[id] = string(serialized)
			}
			s.Require().NoError(rows.Err())
			s.Equal(tc.expected, actual)

			rows, err = s.db.Query(s.ctx, "EXPLAIN (COSTS OFF) "+tc.query, tc.args...)
			s.Require().NoError(err)
			defer rows.Close()

			var planLines []string
			for rows.Next() {
				var line string
				s.Require().NoError(rows.Scan(&line))
				planLines = append(planLines, line)
			}
			s.Require().NoError(rows.Err())
			plan := strings.Join(planLines, "\n")
			s.T().Log(plan)
			s.Contains(plan, "policies_pkey")
			s.NotContains(plan, "policies_id")
		})
	}
}
