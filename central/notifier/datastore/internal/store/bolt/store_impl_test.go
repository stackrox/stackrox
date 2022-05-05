package bolt

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestNotifierStore(t *testing.T) {
	suite.Run(t, new(NotifierStoreTestSuite))
}

type NotifierStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store store.Store
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
	testutils.TearDownDB(suite.db)
}

func (suite *NotifierStoreTestSuite) TestNotifiers() {
	notifiers := []*storage.Notifier{
		{
			Id:           uuid.NewV4().String(),
			Name:         "slack1",
			Type:         "slack",
			LabelDefault: "label1",
		},
		{
			Id:           uuid.NewV4().String(),
			Name:         "pagerduty1",
			Type:         "pagerduty",
			LabelDefault: "label2",
		},
	}

	ctx := sac.WithAllAccess(context.Background())

	// Test Add
	for _, b := range notifiers {
		err := suite.store.Upsert(ctx, b)
		suite.NoError(err)
	}

	for _, b := range notifiers {
		got, exists, err := suite.store.Get(ctx, b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test GetAll
	fromStore, err := suite.store.GetAll(ctx)
	suite.NoError(err)
	suite.Equal(len(notifiers), len(fromStore))
	suite.ElementsMatch(notifiers, fromStore)

	// Test Update
	for _, b := range notifiers {
		b.LabelDefault += "1"
		suite.NoError(suite.store.Upsert(ctx, b))
	}

	for _, b := range notifiers {
		got, exists, err := suite.store.Get(ctx, b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range notifiers {
		suite.NoError(suite.store.Delete(ctx, b.GetId()))
	}

	for _, b := range notifiers {
		_, exists, err := suite.store.Get(ctx, b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
