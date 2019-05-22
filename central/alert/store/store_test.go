package store

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestAlertStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(alertStoreTestSuite))
}

type alertStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (s *alertStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	s.db = db
	s.store = New(db)
}

func (s *alertStoreTestSuite) TearDownSuite() {
	if s.db != nil {
		testutils.TearDownDB(s.db)
	}
}

func (s *alertStoreTestSuite) TestAlerts() {
	alerts := []*storage.Alert{
		{
			Id:             "id1",
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Policy: &storage.Policy{
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			Id:             "id2",
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			Policy: &storage.Policy{
				Severity: storage.Severity_HIGH_SEVERITY,
			},
			State: storage.ViolationState_RESOLVED,
		},
	}

	for _, a := range alerts {
		s.NoError(s.store.AddAlert(a))
	}

	for _, a := range alerts {
		full, exists, err := s.store.GetAlert(a.GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a, full)

		list, exists, err := s.store.ListAlert(a.GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a.GetLifecycleStage(), list.GetLifecycleStage())
		s.Equal(a.GetPolicy().GetSeverity(), list.GetPolicy().GetSeverity())
		s.Equal(a.GetState(), list.GetState())
	}

	for _, a := range alerts {
		a.Policy.Severity = storage.Severity_MEDIUM_SEVERITY
		s.NoError(s.store.UpdateAlert(a))
	}

	for _, a := range alerts {
		full, exists, err := s.store.GetAlert(a.GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a, full)

		list, exists, err := s.store.ListAlert(a.GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a.GetLifecycleStage(), list.GetLifecycleStage())
		s.Equal(a.GetPolicy().GetSeverity(), list.GetPolicy().GetSeverity())
		s.Equal(a.GetState(), list.GetState())
	}

}
