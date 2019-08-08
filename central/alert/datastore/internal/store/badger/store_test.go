package badger

import (
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestAlertStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(alertStoreTestSuite))
}

type alertStoreTestSuite struct {
	suite.Suite

	db  *badger.DB
	dir string

	store store.Store
}

func (s *alertStoreTestSuite) SetupSuite() {
	db, dir, err := badgerhelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	s.db = db
	s.dir = dir
	s.store = New(db)
}

func (s *alertStoreTestSuite) TearDownSuite() {
	if s.db != nil {
		testutils.TearDownBadger(s.db, s.dir)
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
