package boltdb

import (
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
	db, err := boltFromTmpDir()
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
			Name:         "slack1",
			Type:         "slack",
			LabelDefault: "label1",
		},
		{
			Name:         "pagerduty1",
			Type:         "pagerduty",
			LabelDefault: "label2",
		},
	}

	// Test Add
	for _, b := range notifiers {
		id, err := suite.AddNotifier(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, b := range notifiers {
		got, exists, err := suite.GetNotifier(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Update
	for _, b := range notifiers {
		b.LabelDefault += "1"
		suite.NoError(suite.UpdateNotifier(b))
	}

	for _, b := range notifiers {
		got, exists, err := suite.GetNotifier(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range notifiers {
		suite.NoError(suite.RemoveNotifier(b.GetId()))
	}

	for _, b := range notifiers {
		_, exists, err := suite.GetNotifier(b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
