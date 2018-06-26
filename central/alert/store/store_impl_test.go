package store

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
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
	alert1 := &v1.Alert{
		Id: "id1",
		Policy: &v1.Policy{
			Severity: v1.Severity_LOW_SEVERITY,
		},
	}
	err := suite.store.AddAlert(alert1)
	suite.Nil(err)
	alert2 := &v1.Alert{
		Id: "id2",
		Policy: &v1.Policy{
			Severity: v1.Severity_HIGH_SEVERITY,
		},
	}
	err = suite.store.AddAlert(alert2)
	suite.Nil(err)

	// Get all alerts
	alerts, err := suite.store.GetAlerts(&v1.ListAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	count, err := suite.store.CountAlerts()
	suite.Nil(err)
	suite.Equal(2, count)

	alert1.Policy.Severity = v1.Severity_HIGH_SEVERITY
	suite.store.UpdateAlert(alert1)
	alerts, err = suite.store.GetAlerts(&v1.ListAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	suite.store.RemoveAlert(alert1.Id)
	alerts, err = suite.store.GetAlerts(&v1.ListAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert2}, alerts)
}
