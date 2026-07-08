//go:build sql_integration

package collation

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestCollationReconciliation(t *testing.T) {
	suite.Run(t, new(CollationSuite))
}

type CollationSuite struct {
	suite.Suite
	testDB *pgtest.TestPostgres
	ctx    context.Context
}

func (s *CollationSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *CollationSuite) TestCheckMismatch_NoMismatch() {
	recorded, actual, mismatch, err := CheckMismatch(s.ctx, s.testDB.DB)
	s.NoError(err)
	s.False(mismatch, "fresh database should not have a collation mismatch")
	// On a healthy database both versions should be non-empty and equal.
	if recorded != "" {
		s.Equal(recorded, actual)
	}
}

func (s *CollationSuite) TestAffectedIndexes_FiltersCorrectly() {
	db := s.testDB.DB

	// Create a test table with columns of different types.
	_, err := db.Exec(s.ctx, `CREATE TABLE IF NOT EXISTS collation_test (
		id serial PRIMARY KEY,
		name text NOT NULL,
		count integer NOT NULL
	)`)
	s.Require().NoError(err)

	// BTREE index on text column — should be included.
	_, err = db.Exec(s.ctx, `CREATE INDEX IF NOT EXISTS collation_test_name_idx ON collation_test (name)`)
	s.Require().NoError(err)

	// BTREE index on integer column — should be excluded (indcollation = 0).
	_, err = db.Exec(s.ctx, `CREATE INDEX IF NOT EXISTS collation_test_count_idx ON collation_test (count)`)
	s.Require().NoError(err)

	// HASH index on text column — should be excluded (not BTREE).
	_, err = db.Exec(s.ctx, `CREATE INDEX IF NOT EXISTS collation_test_name_hash ON collation_test USING hash (name)`)
	s.Require().NoError(err)

	s.T().Cleanup(func() {
		_, _ = db.Exec(s.ctx, `DROP TABLE IF EXISTS collation_test CASCADE`)
	})

	indexes, err := AffectedIndexes(s.ctx, db)
	s.Require().NoError(err)

	// Find our test indexes in the results.
	var foundTextBtree, foundIntBtree, foundTextHash bool
	for _, idx := range indexes {
		switch idx.Name {
		case "collation_test_name_idx":
			foundTextBtree = true
		case "collation_test_count_idx":
			foundIntBtree = true
		case "collation_test_name_hash":
			foundTextHash = true
		}
	}

	s.True(foundTextBtree, "BTREE index on text column should be included")
	s.False(foundIntBtree, "BTREE index on integer column should be excluded")
	s.False(foundTextHash, "HASH index on text column should be excluded")
}

func (s *CollationSuite) TestReconcile_NoMismatch_Noop() {
	err := Reconcile(s.ctx, s.testDB.DB, 30*time.Second)
	s.NoError(err, "Reconcile should be a no-op when no mismatch exists")
}

// TestReconcile_ReindexesAndRefreshes verifies Reconcile executes without error
// on a healthy database. Full mismatch testing requires a multi-glibc environment
// (e.g., initdb under glibc 2.28, then run under glibc 2.34).
func (s *CollationSuite) TestReconcile_ReindexesAndRefreshes() {
	err := Reconcile(s.ctx, s.testDB.DB, 30*time.Second)
	s.NoError(err)

	// Verify no mismatch after reconcile.
	_, _, mismatch, err := CheckMismatch(s.ctx, s.testDB.DB)
	s.NoError(err)
	s.False(mismatch)
}
