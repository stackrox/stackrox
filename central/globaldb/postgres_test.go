//go:build sql_integration

package globaldb

import (
	"context"
	"testing"

	"github.com/pkg/errors"
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
