package boltdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltAlerts(t *testing.T) {
	suite.Run(t, new(BoltAlertsTestSuite))
}

type BoltAlertsTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltAlertsTestSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := MakeBoltDB(tmpDir)
	if err != nil {
		fmt.Printf("Error making BoltDB: %+v", err)
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db.(*BoltDB)
}

func (suite *BoltAlertsTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltAlertsTestSuite) TestAlerts() {
	alert1 := &v1.Alert{
		Id:       "id1",
		Severity: v1.Severity_LOW_SEVERITY,
	}
	err := suite.AddAlert(alert1)
	suite.Nil(err)
	alert2 := &v1.Alert{
		Id:       "id2",
		Severity: v1.Severity_HIGH_SEVERITY,
	}
	err = suite.AddAlert(alert2)
	suite.Nil(err)

	// Get all alerts
	alerts, err := suite.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	alert1.Severity = v1.Severity_HIGH_SEVERITY
	suite.UpdateAlert(alert1)
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	suite.RemoveAlert(alert1.Id)
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert2}, alerts)
}
