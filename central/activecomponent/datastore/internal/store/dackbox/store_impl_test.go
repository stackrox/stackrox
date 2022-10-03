package dackbox

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/activecomponent/converter"
	acDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestActiveComponentStore(t *testing.T) {
	suite.Run(t, new(ActiveComponentStoreTestSuite))
}

type ActiveComponentStoreTestSuite struct {
	suite.Suite

	ctx   context.Context
	db    *rocksdb.RocksDB
	dacky *dackbox.DackBox
	store store.Store
}

func (suite *ActiveComponentStoreTestSuite) SetupSuite() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())

	var err error
	suite.dacky, err = dackbox.NewRocksDBDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNow("failed to create counter", err.Error())
	}
	suite.store = New(suite.dacky, concurrency.NewKeyFence())
	suite.ctx = sac.WithAllAccess(context.Background())
}

func (suite *ActiveComponentStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ActiveComponentStoreTestSuite) TestUpsertDelete() {
	deployments := []string{"dep0", "dep1", "dep2"}
	imageComponents := []string{"component0", "component1", "component2"}

	testCases := []struct {
		name           string
		upserts        set.Set[int]
		containerNames []string
		deletes        set.Set[int]
	}{
		{
			name:           "Add all delete all from empty",
			upserts:        set.NewSet(0, 1, 2, 3, 4, 5, 6, 7, 8),
			containerNames: []string{"container0"},
			deletes:        set.NewSet(0, 1, 2, 3, 4, 5, 6, 7, 8),
		},
		{
			name:           "Add even delete odd",
			upserts:        set.NewSet(0, 2, 4, 6, 8),
			containerNames: []string{"container1"},
			deletes:        set.NewSet(1, 3, 5, 7),
		},
		{
			name:           "Add none delete none",
			upserts:        set.NewSet[int](),
			containerNames: []string{"container2"},
			deletes:        set.NewSet[int](),
		},
		{
			name:           "Add old delete 1 and 3",
			upserts:        set.NewSet(1, 3, 5, 7),
			containerNames: []string{"container2"},
			deletes:        set.NewSet(1, 3),
		},
		{
			name:           "Add all delete all",
			upserts:        set.NewSet(0, 1, 2, 3, 4, 5, 6, 7, 8),
			containerNames: []string{"container4", "container5"},
			deletes:        set.NewSet(0, 1, 2, 3, 4, 5, 6, 7),
		},
	}
	expectedMapToContainerNames := make(map[int][]string)
	for _, testCase := range testCases {
		suite.T().Run(testCase.name, func(t *testing.T) {
			var acs []*storage.ActiveComponent
			activeContext := make(map[string]*storage.ActiveComponent_ActiveContext)
			for _, containerName := range testCase.containerNames {
				activeContext[containerName] = &storage.ActiveComponent_ActiveContext{
					ContainerName: containerName,
				}
			}
			for upsert := range testCase.upserts {
				deploymentID := deployments[upsert/3]
				componentID := imageComponents[upsert%3]
				ac := &storage.ActiveComponent{
					Id:                  converter.ComposeID(deploymentID, componentID),
					DeploymentId:        deploymentID,
					ComponentId:         componentID,
					ActiveContextsSlice: converter.ConvertActiveContextsMapToSlice(activeContext),
				}
				acs = append(acs, ac)
				expectedMapToContainerNames[upsert] = testCase.containerNames
			}
			suite.Assert().NoError(suite.store.UpsertMany(suite.ctx, acs))
			suite.verify(deployments, imageComponents, expectedMapToContainerNames)
			var ids []string
			for del := range testCase.deletes {
				deploymentID := deployments[del/3]
				componentID := imageComponents[del%3]
				ids = append(ids, converter.ComposeID(deploymentID, componentID))
				delete(expectedMapToContainerNames, del)
			}
			suite.Assert().NoError(suite.store.DeleteMany(suite.ctx, ids))
			suite.verify(deployments, imageComponents, expectedMapToContainerNames)
		})
	}
}

func (suite *ActiveComponentStoreTestSuite) verify(deployments, imageComponents []string, expectedMap map[int][]string) {
	dackTxn, err := suite.dacky.NewReadOnlyTransaction()
	suite.Assert().NoError(err)
	defer dackTxn.Discard()

	count, err := acDackBox.Reader.CountIn(acDackBox.Bucket, dackTxn)
	suite.Assert().NoError(err)
	suite.Assert().Len(expectedMap, count)

	for expected, containerNames := range expectedMap {
		deploymentID := deployments[expected/3]
		componentID := imageComponents[expected%3]
		id := converter.ComposeID(deploymentID, componentID)
		ac, exist, err := suite.store.Get(suite.ctx, id)
		suite.Assert().NoError(err)
		suite.Assert().True(exist)
		suite.Assert().Len(ac.GetActiveContextsSlice(), len(containerNames))
		for _, context := range ac.GetActiveContextsSlice() {
			suite.Assert().Contains(containerNames, context.ContainerName)
		}
		tos := dackTxn.Graph().GetRefsToPrefix(componentDackBox.BucketHandler.GetKey(componentID), acDackBox.Bucket)
		suite.Assert().Contains(tos, acDackBox.BucketHandler.GetKey(id))
		froms := dackTxn.Graph().GetRefsFromPrefix(deploymentDackBox.BucketHandler.GetKey(deploymentID), acDackBox.Bucket)
		suite.Assert().Contains(froms, acDackBox.BucketHandler.GetKey(id))
	}
}
