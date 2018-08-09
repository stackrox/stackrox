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
	suite.Run(t, new(AlertStoreTestSuite))
}

type AlertStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *AlertStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *AlertStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *AlertStoreTestSuite) TestAlerts() {
	alerts := []*v1.Alert{
		{
			Id: "id1",
			Policy: &v1.Policy{
				Severity: v1.Severity_LOW_SEVERITY,
			},
		},
		{
			Id: "id2",
			Policy: &v1.Policy{
				Severity: v1.Severity_HIGH_SEVERITY,
			},
		},
	}

	for _, a := range alerts {
		suite.NoError(suite.store.AddAlert(a))
	}

	retrievedAlerts, err := suite.store.GetAlerts()
	suite.NoError(err)
	suite.ElementsMatch(alerts, retrievedAlerts)

	for _, a := range alerts {
		full, exists, err := suite.store.GetAlert(a.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(a, full)

		list, exists, err := suite.store.ListAlert(a.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(a.GetPolicy().GetSeverity(), list.GetPolicy().GetSeverity())
	}

	for _, a := range alerts {
		a.Policy.Severity = v1.Severity_MEDIUM_SEVERITY
		suite.NoError(suite.store.UpdateAlert(a))
	}

	for _, a := range alerts {
		full, exists, err := suite.store.GetAlert(a.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(a, full)

		list, exists, err := suite.store.ListAlert(a.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(a.GetPolicy().GetSeverity(), list.GetPolicy().GetSeverity())
	}

}
