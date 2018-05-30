package boltdb

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltAlerts(t *testing.T) {
	suite.Run(t, new(BoltAlertsTestSuite))
}

type BoltAlertsTestSuite struct {
	suite.Suite
	*BoltDB
}

func boltFromTmpDir() (*BoltDB, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return New(filepath.Join(tmpDir, "prevent.db"))
}

func (suite *BoltAlertsTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltAlertsTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltAlertsTestSuite) TestAlerts() {
	alert1 := &v1.Alert{
		Id: "id1",
		Policy: &v1.Policy{
			Severity: v1.Severity_LOW_SEVERITY,
		},
	}
	err := suite.AddAlert(alert1)
	suite.Nil(err)
	alert2 := &v1.Alert{
		Id: "id2",
		Policy: &v1.Policy{
			Severity: v1.Severity_HIGH_SEVERITY,
		},
	}
	err = suite.AddAlert(alert2)
	suite.Nil(err)

	// Get all alerts
	alerts, err := suite.GetAlerts(&v1.ListAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	count, err := suite.CountAlerts()
	suite.Nil(err)
	suite.Equal(2, count)

	alert1.Policy.Severity = v1.Severity_HIGH_SEVERITY
	suite.UpdateAlert(alert1)
	alerts, err = suite.GetAlerts(&v1.ListAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	suite.RemoveAlert(alert1.Id)
	alerts, err = suite.GetAlerts(&v1.ListAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert2}, alerts)
}
