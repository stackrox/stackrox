package store

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"go.etcd.io/bbolt"
)

func TestTelemetryStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(telemetryStoreTestSuite))
}

type telemetryStoreTestSuite struct {
	suite.Suite

	db *bbolt.DB

	store Store
}

func (s *telemetryStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	s.db = db
	s.store, err = New(db)
	s.Require().NoError(err)
}

func (s *telemetryStoreTestSuite) TearDownSuite() {
	if s.db != nil {
		testutils.TearDownDB(s.db)
	}
}

func (s *telemetryStoreTestSuite) TestTelemetryStore() {
	configOn := &storage.TelemetryConfiguration{
		Enabled: true,
	}
	configOff := &storage.TelemetryConfiguration{
		Enabled: false,
	}

	config, err := s.store.GetTelemetryConfig()
	s.NoError(err)
	s.Nil(config)

	s.NoError(s.store.SetTelemetryConfig(configOn))
	config, err = s.store.GetTelemetryConfig()
	s.NoError(err)
	s.Equal(configOn, config)

	s.NoError(s.store.SetTelemetryConfig(configOff))
	config, err = s.store.GetTelemetryConfig()
	s.NoError(err)
	s.Equal(configOff, config)
}
