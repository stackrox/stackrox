package badger

import (
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/pod/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestPodStore(t *testing.T) {
	suite.Run(t, new(PodStoreTestSuite))
}

type PodStoreTestSuite struct {
	suite.Suite

	db  *badger.DB
	dir string

	store store.Store
}

func (suite *PodStoreTestSuite) SetupSuite() {
	db, dir, err := badgerhelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BadgerDB", err.Error())
	}

	suite.db = db
	suite.dir = dir
	suite.store = New(db)
}

func (suite *PodStoreTestSuite) TearDownSuite() {
	testutils.TearDownBadger(suite.db, suite.dir)
}

func (suite *PodStoreTestSuite) verifyPodsAre(store store.Store, pods ...*storage.Pod) {
	for _, p := range pods {
		got, exists, err := store.GetPod(p.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(p, got)
	}

	// Test Count
	count, err := store.CountPods()
	suite.NoError(err)
	suite.Equal(len(pods), count)
}

func (suite *PodStoreTestSuite) TestPods() {
	pods := []*storage.Pod{
		{
			Id:           "fooID",
			DeploymentId: "deployID",
			Active:       false,
			Instances: []*storage.ContainerInstance{
				{
					ContainerName: "fooContainer",
				},
			},
		},
		{
			Id:           "barID",
			DeploymentId: "deployID",
			Active:       true,
			Instances: []*storage.ContainerInstance{
				{
					ContainerName: "barContainer",
				},
			},
		},
	}

	// Test Upsert
	for _, d := range pods {
		suite.NoError(suite.store.UpsertPod(d))
	}

	suite.verifyPodsAre(suite.store, pods...)

	// This verifies that things work as expected on restarts.
	newStore := New(suite.db)

	suite.verifyPodsAre(newStore, pods...)

	// Test Remove
	for _, d := range pods {
		suite.NoError(suite.store.RemovePod(d.GetId()))
	}

	suite.verifyPodsAre(suite.store)

	newStore = New(suite.db)

	suite.verifyPodsAre(newStore)
}
