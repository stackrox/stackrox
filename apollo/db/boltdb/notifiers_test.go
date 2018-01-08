package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltNotifiers(t *testing.T) {
	suite.Run(t, new(BoltNotifierTestSuite))
}

type BoltNotifierTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltNotifierTestSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := New(tmpDir)
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltNotifierTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltNotifierTestSuite) TestNotifiers() {
	notifiers := []*v1.Notifier{
		{
			Name:   "slack1",
			Type:   "slack",
			Config: map[string]string{"username": "srox"},
		},
		{
			Name:   "pagerduty1",
			Type:   "pagerduty",
			Config: map[string]string{"username": "srox"},
		},
	}

	// Test Add
	for _, b := range notifiers {
		suite.NoError(suite.AddNotifier(b))
	}

	for _, b := range notifiers {
		got, exists, err := suite.GetNotifier(b.Name)
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Update
	for _, b := range notifiers {
		b.Config["newparam"] = "value"
		suite.NoError(suite.UpdateNotifier(b))
	}

	for _, b := range notifiers {
		got, exists, err := suite.GetNotifier(b.GetName())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range notifiers {
		suite.NoError(suite.RemoveNotifier(b.GetName()))
	}

	for _, b := range notifiers {
		_, exists, err := suite.GetNotifier(b.GetName())
		suite.NoError(err)
		suite.False(exists)
	}
}
