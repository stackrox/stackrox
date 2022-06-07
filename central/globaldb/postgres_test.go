package globaldb

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type PostgresUtilitySuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestConfigSetup(t *testing.T) {
	suite.Run(t, new(PostgresUtilitySuite))
}

func (s *PostgresUtilitySuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

}

func (s *PostgresUtilitySuite) TearDownTest() {
	s.envIsolator.RestoreAll()
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
		sourceMap, err := ParseSource(c.source)
		if c.err != nil && err != nil {
			s.Equal(c.err.Error(), err.Error())
		} else if c.err != nil || err != nil {
			s.Fail("expected error does not equal error received")
		}

		s.Equal(c.expectedMap, sourceMap)
	}

}
