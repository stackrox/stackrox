//go:build sql_integration

package globaldb

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/central/globaldb/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

type PostgresUtilitySuite struct {
	suite.Suite
}

func TestConfigSetup(t *testing.T) {
	suite.Run(t, new(PostgresUtilitySuite))
}

func (s *PostgresUtilitySuite) TestSourceParser() {
	cases := []struct {
		name        string
		source      string
		expectedMap map[string]string
		err         error
	}{
		{
			name:        "Case 1",
			source:      "",
			expectedMap: nil,
			err:         errors.New("source string is empty"),
		},
		{
			name:   "Case 2",
			source: "host=testHost port=5432 database=testDB sensitiveField=testSensitive",
			expectedMap: map[string]string{
				"host":           "testHost",
				"port":           "5432",
				"database":       "testDB",
				"sensitiveField": "testSensitive",
			},
			err: nil,
		},
		{
			name:   "Case 3",
			source: "host=testHost  port=5432  database=testDB   sensitiveField=testSensitive  ",
			expectedMap: map[string]string{
				"host":           "testHost",
				"port":           "5432",
				"database":       "testDB",
				"sensitiveField": "testSensitive",
			},
			err: nil,
		},
		{
			name:   "Case 4",
			source: "host=testHost port=5432 database=testDB sensitiveField=testWith=InValue",
			expectedMap: map[string]string{
				"host":           "testHost",
				"port":           "5432",
				"database":       "testDB",
				"sensitiveField": "testWith=InValue",
			},
			err: nil,
		},
	}

	for _, c := range cases {
		log.Info(c.name)
		sourceMap, err := pgconfig.ParseSource(c.source)
		if c.err != nil && err != nil {
			s.Equal(c.err.Error(), err.Error())
		} else if c.err != nil || err != nil {
			s.Fail("expected error does not equal error received")
		}

		s.Equal(c.expectedMap, sourceMap)
	}

}

func (s *PostgresUtilitySuite) TestCollectPostgresStats() {
	ctx := sac.WithAllAccess(context.Background())
	tp := pgtest.ForT(s.T())

	stats := CollectPostgresStats(ctx, tp.DB)
	s.NotNil(stats)
	s.Equal(true, stats.DatabaseAvailable)
	s.True(len(stats.Tables) > 0)

	tp.Close()

	stats = CollectPostgresStats(ctx, tp.DB)
	s.NotNil(stats)
	s.Equal(false, stats.DatabaseAvailable)
}

func (s *PostgresUtilitySuite) TestCollectPostgresIndexStats() {
	ctx := sac.WithAllAccess(context.Background())
	tp := pgtest.ForT(s.T())
	defer tp.Close()

	// On a clean test DB there should be no invalid indexes.
	CollectPostgresIndexStats(ctx, tp.DB)
	s.Equal(0, testutil.CollectAndCount(metrics.PostgresInvalidIndexes))

	// Create a table and index, then mark the index invalid.
	_, err := tp.DB.Exec(ctx, "CREATE TABLE test_invalid_idx_check (id int)")
	s.Require().NoError(err)
	_, err = tp.DB.Exec(ctx, "CREATE INDEX test_idx ON test_invalid_idx_check (id)")
	s.Require().NoError(err)
	_, err = tp.DB.Exec(ctx, "UPDATE pg_index SET indisvalid = false WHERE indexrelid = 'test_idx'::regclass")
	s.Require().NoError(err)

	CollectPostgresIndexStats(ctx, tp.DB)
	s.Equal(1, testutil.CollectAndCount(metrics.PostgresInvalidIndexes))
	s.Equal(1.0, testutil.ToFloat64(metrics.PostgresInvalidIndexes.With(prometheus.Labels{
		"index_name": "test_idx",
		"table_name": "test_invalid_idx_check",
	})))

	// Clean up: restore validity so DROP works cleanly.
	_, err = tp.DB.Exec(ctx, "UPDATE pg_index SET indisvalid = true WHERE indexrelid = 'test_idx'::regclass")
	s.Require().NoError(err)
	_, err = tp.DB.Exec(ctx, "DROP TABLE test_invalid_idx_check")
	s.Require().NoError(err)

	CollectPostgresIndexStats(ctx, tp.DB)
	s.Equal(0, testutil.CollectAndCount(metrics.PostgresInvalidIndexes))
}
