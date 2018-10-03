package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
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
		s.db.Close()
		os.Remove(s.db.Path())
	}
}

func (s *alertStoreTestSuite) TestAlerts() {
	alerts := []*v1.Alert{
		{
			Id:             "id1",
			LifecycleStage: v1.LifecycleStage_RUN_TIME,
			Policy: &v1.Policy{
				Severity: v1.Severity_LOW_SEVERITY,
			},
		},
		{
			Id:             "id2",
			LifecycleStage: v1.LifecycleStage_DEPLOY_TIME,
			Policy: &v1.Policy{
				Severity: v1.Severity_HIGH_SEVERITY,
			},
		},
	}

	for _, a := range alerts {
		s.NoError(s.store.AddAlert(a))
	}

	retrievedAlerts, err := s.store.GetAlerts()
	s.NoError(err)
	s.ElementsMatch(alerts, retrievedAlerts)

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
	}

	for _, a := range alerts {
		a.Policy.Severity = v1.Severity_MEDIUM_SEVERITY
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
	}

}
