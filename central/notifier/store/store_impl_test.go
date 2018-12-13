package store

import (
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestNotifierStore(t *testing.T) {
	suite.Run(t, new(NotifierStoreTestSuite))
}

type NotifierStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *NotifierStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *NotifierStoreTestSuite) TearDownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *NotifierStoreTestSuite) TestNotifiers() {
	notifiers := []*storage.Notifier{
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
		id, err := suite.store.AddNotifier(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, b := range notifiers {
		got, exists, err := suite.store.GetNotifier(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Update
	for _, b := range notifiers {
		b.LabelDefault += "1"
		suite.NoError(suite.store.UpdateNotifier(b))
	}

	for _, b := range notifiers {
		got, exists, err := suite.store.GetNotifier(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range notifiers {
		suite.NoError(suite.store.RemoveNotifier(b.GetId()))
	}

	for _, b := range notifiers {
		_, exists, err := suite.store.GetNotifier(b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
