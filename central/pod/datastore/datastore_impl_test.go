package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	searcherMocks "github.com/stackrox/stackrox/central/pod/datastore/internal/search/mocks"
	indexerMocks "github.com/stackrox/stackrox/central/pod/index/mocks"
	storeMocks "github.com/stackrox/stackrox/central/pod/store/mocks"
	indicatorMocks "github.com/stackrox/stackrox/central/processindicator/datastore/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/process/filter"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	ctx         = sac.WithAllAccess(context.Background())
	expectedPod = fixtures.GetPod()
)

func TestPodDatastoreSuite(t *testing.T) {
	suite.Run(t, new(PodDataStoreTestSuite))
}

type PodDataStoreTestSuite struct {
	suite.Suite

	datastore *datastoreImpl

	storage      *storeMocks.MockStore
	indexer      *indexerMocks.MockIndexer
	searcher     *searcherMocks.MockSearcher
	processStore *indicatorMocks.MockDataStore
	filter       filter.Filter

	mockCtrl *gomock.Controller
}

func (suite *PodDataStoreTestSuite) SetupTest() {
	mockCtrl := gomock.NewController(suite.T())
	suite.mockCtrl = mockCtrl
	suite.storage = storeMocks.NewMockStore(mockCtrl)
	suite.indexer = indexerMocks.NewMockIndexer(mockCtrl)
	suite.searcher = searcherMocks.NewMockSearcher(mockCtrl)
	suite.processStore = indicatorMocks.NewMockDataStore(mockCtrl)
	suite.filter = filter.NewFilter(5, []int{5, 4, 3, 2, 1})

	var err error
	if !features.PostgresDatastore.Enabled() {
		suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)
		suite.storage.EXPECT().GetKeysToIndex(ctx).Return(nil, nil)
	}
	suite.datastore, err = newDatastoreImpl(ctx, suite.storage, suite.indexer, suite.searcher, suite.processStore, suite.filter)
	suite.NoError(err)
}

func (suite *PodDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PodDataStoreTestSuite) TestNoAccessAllowed() {
	ctx := sac.WithNoAccess(context.Background())

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	_, ok, _ := suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.False(ok)

	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "permission denied")

	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "permission denied")
}

func (suite *PodDataStoreTestSuite) TestSearch() {
	suite.searcher.EXPECT().Search(ctx, nil).Return(nil, nil)
	_, err := suite.datastore.Search(ctx, nil)
	suite.NoError(err)
}

func (suite *PodDataStoreTestSuite) TestGetPod() {
	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	pod, ok, err := suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.NoError(err)
	suite.True(ok)
	suite.Equal(expectedPod, pod)

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	_, ok, err = suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.NoError(err)
	suite.False(ok)

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, errors.New("error"))
	_, _, err = suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.Error(err, "error")
}

func (suite *PodDataStoreTestSuite) TestUpsertPodNew() {
	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, expectedPod.GetId()).Return(nil)
	suite.NoError(suite.datastore.UpsertPod(ctx, expectedPod))

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, expectedPod.GetId()).Return(nil)
	suite.NoError(suite.datastore.UpsertPod(ctx, expectedPod))
}

func (suite *PodDataStoreTestSuite) TestUpsertPodExists() {
	// Renaming to make things clear
	oldPod := expectedPod

	pod := fixtures.GetPod()
	pod.TerminatedInstances = make([]*storage.Pod_ContainerInstanceList, 0)
	// Update one instance.
	pod.LiveInstances[0] = &storage.ContainerInstance{
		InstanceId:    pod.LiveInstances[0].InstanceId,
		ContainerName: pod.LiveInstances[0].ContainerName,
		ImageDigest:   "sha256:3984274924983274198",
	}
	// Terminate the other instance.
	terminatedInst0 := &storage.ContainerInstance{
		InstanceId:    pod.LiveInstances[1].InstanceId,
		ContainerName: pod.LiveInstances[1].ContainerName,
		Finished: &types.Timestamp{
			Seconds: 10,
		},
		ExitCode:          0,
		TerminationReason: "Completed",
	}
	pod.LiveInstances[1] = terminatedInst0
	// Add a new terminated instance.
	terminatedInst1 := &storage.ContainerInstance{
		InstanceId: &storage.ContainerInstanceID{
			Id: "newdeadcontainerid",
		},
		ContainerName: "newdeadcontainername",
		Finished: &types.Timestamp{
			Seconds: 9,
		},
		ExitCode:          137,
		TerminationReason: "Error",
	}
	pod.LiveInstances = append(pod.LiveInstances, terminatedInst1)
	// Add a new live instance.
	liveInst := &storage.ContainerInstance{
		InstanceId: &storage.ContainerInstanceID{
			Id: "newlivecontainerid",
		},
		ContainerName: "newlivecontainername",
		Started: &types.Timestamp{
			Seconds: 8,
		},
	}
	pod.LiveInstances = append(pod.LiveInstances, liveInst)

	// merged should have all the previously dead instances plus the two new ones
	// as well as the new live instances.
	// This is the pod we expect to actually upsert to the DB.
	merged := fixtures.GetPod()
	merged.LiveInstances = []*storage.ContainerInstance{pod.LiveInstances[0], pod.LiveInstances[3]}
	merged.TerminatedInstances[1].Instances = append(merged.TerminatedInstances[1].Instances, terminatedInst0)
	merged.TerminatedInstances = append(merged.TerminatedInstances, &storage.Pod_ContainerInstanceList{
		Instances: []*storage.ContainerInstance{terminatedInst1},
	})
	suite.storage.EXPECT().Get(ctx, pod.GetId()).Return(oldPod, true, nil)
	suite.storage.EXPECT().Upsert(ctx, merged).Return(nil)
	suite.indexer.EXPECT().AddPod(merged).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, merged.GetId()).Return(nil)
	suite.NoError(suite.datastore.UpsertPod(ctx, pod))
}

func (suite *PodDataStoreTestSuite) TestRemovePod() {
	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(nil)
	suite.indexer.EXPECT().DeletePod(expectedPod.GetId()).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, expectedPod.GetId()).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsByPod(gomock.Any(), expectedPod.GetId())
	suite.NoError(suite.datastore.RemovePod(ctx, expectedPod.GetId()))

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, false, nil)
	suite.NoError(suite.datastore.RemovePod(ctx, expectedPod.GetId()))

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, false, errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()))

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(nil)
	suite.indexer.EXPECT().DeletePod(expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(nil)
	suite.indexer.EXPECT().DeletePod(expectedPod.GetId()).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")
}

func (suite *PodDataStoreTestSuite) TestReconciliationFullReindex() {
	if features.PostgresDatastore.Enabled() {
		return
	}
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(true, nil)

	pod1 := fixtures.GetPod()
	pod1.Id = "A"
	pod2 := fixtures.GetPod()
	pod2.Id = "B"

	suite.storage.EXPECT().GetIDs(ctx).Return([]string{"A", "B", "C"}, nil)
	suite.storage.EXPECT().GetMany(ctx, []string{"A", "B", "C"}).Return([]*storage.Pod{pod1, pod2}, nil, nil)
	suite.indexer.EXPECT().AddPods([]*storage.Pod{pod1, pod2}).Return(nil)

	suite.storage.EXPECT().GetKeysToIndex(ctx).Return([]string{"D", "E"}, nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, []string{"D", "E"}).Return(nil)

	suite.indexer.EXPECT().MarkInitialIndexingComplete().Return(nil)

	// Create a new data store to trigger the reindexing.
	_, err := newDatastoreImpl(ctx, suite.storage, suite.indexer, nil, suite.processStore, suite.filter)
	suite.NoError(err)
}

func (suite *PodDataStoreTestSuite) TestReconciliationPartialReindex() {
	if features.PostgresDatastore.Enabled() {
		return
	}
	suite.storage.EXPECT().GetKeysToIndex(ctx).Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	pod1 := fixtures.GetPod()
	pod1.Id = "A"
	pod2 := fixtures.GetPod()
	pod2.Id = "B"
	pod3 := fixtures.GetPod()
	pod3.Id = "C"

	podList := []*storage.Pod{pod1, pod2, pod3}

	suite.storage.EXPECT().GetMany(ctx, []string{"A", "B", "C"}).Return(podList, nil, nil)
	suite.indexer.EXPECT().AddPods(podList).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, []string{"A", "B", "C"}).Return(nil)

	_, err := newDatastoreImpl(ctx, suite.storage, suite.indexer, nil, suite.processStore, suite.filter)
	suite.NoError(err)

	// Make podList just A,B so C should be deleted
	podList = podList[:1]
	suite.storage.EXPECT().GetKeysToIndex(ctx).Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.storage.EXPECT().GetMany(ctx, []string{"A", "B", "C"}).Return(podList, []int{2}, nil)
	suite.indexer.EXPECT().AddPods(podList).Return(nil)
	suite.indexer.EXPECT().DeletePods([]string{"C"}).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(ctx, []string{"A", "B", "C"}).Return(nil)

	_, err = newDatastoreImpl(ctx, suite.storage, suite.indexer, nil, suite.processStore, suite.filter)
	suite.NoError(err)
}
