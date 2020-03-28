package datastore

import (
	"context"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	searcherMocks "github.com/stackrox/rox/central/pod/datastore/internal/search/mocks"
	indexerMocks "github.com/stackrox/rox/central/pod/index/mocks"
	storeMocks "github.com/stackrox/rox/central/pod/store/mocks"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
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
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)
	suite.storage.EXPECT().GetKeysToIndex().Return(nil, nil)
	suite.datastore, err = newDatastoreImpl(suite.storage, suite.indexer, suite.searcher, suite.processStore, suite.filter)
	suite.NoError(err)
}

func (suite *PodDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PodDataStoreTestSuite) TestNoAccessAllowed() {
	ctx := sac.WithNoAccess(context.Background())

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, true, nil)
	_, ok, _ := suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.False(ok)

	_, err := suite.datastore.GetPods(ctx, []string{expectedPod.GetId()})
	suite.Error(err, "permission denied")

	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "permission denied")

	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "permission denied")
}

func (suite *PodDataStoreTestSuite) TestSearch() {
	suite.searcher.EXPECT().Search(ctx, nil).Return(nil, nil)
	_, err := suite.datastore.Search(ctx, nil)
	suite.NoError(err)
}

func (suite *PodDataStoreTestSuite) TestGetPod() {
	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, true, nil)
	pod, ok, err := suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.NoError(err)
	suite.True(ok)
	suite.Equal(expectedPod, pod)

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, nil)
	_, ok, err = suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.NoError(err)
	suite.False(ok)

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, errors.New("error"))
	_, _, err = suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.Error(err, "error")
}

func (suite *PodDataStoreTestSuite) TestGetPods() {
	suite.storage.EXPECT().GetPodsWithIDs(expectedPod.GetId()).Return([]*storage.Pod{expectedPod}, []int{0}, nil)
	pods, err := suite.datastore.GetPods(ctx, []string{expectedPod.GetId()})
	suite.NoError(err)
	suite.Equal([]*storage.Pod{expectedPod}, pods)

	suite.storage.EXPECT().GetPodsWithIDs(expectedPod.GetId()).Return(nil, nil, nil)
	_, err = suite.datastore.GetPods(ctx, []string{expectedPod.GetId()})
	suite.NoError(err)

	suite.storage.EXPECT().GetPodsWithIDs(expectedPod.GetId()).Return(nil, nil, errors.New("error"))
	_, err = suite.datastore.GetPods(ctx, []string{expectedPod.GetId()})
	suite.Error(err, "error")
}

func (suite *PodDataStoreTestSuite) TestCountPods() {
	suite.storage.EXPECT().CountPods().Return(2, nil)
	numPods, err := suite.datastore.CountPods(ctx)
	suite.NoError(err)
	suite.Equal(numPods, 2)
}

func (suite *PodDataStoreTestSuite) TestUpsertPodNew() {
	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().UpsertPod(expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(expectedPod.GetId()).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsOfStaleContainersByPod(gomock.Any(), expectedPod).Return(nil)
	suite.NoError(suite.datastore.UpsertPod(ctx, expectedPod))

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().UpsertPod(expectedPod).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().UpsertPod(expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().UpsertPod(expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().UpsertPod(expectedPod).Return(nil)
	suite.indexer.EXPECT().AddPod(expectedPod).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(expectedPod.GetId()).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsOfStaleContainersByPod(gomock.Any(), expectedPod).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod))
}

func (suite *PodDataStoreTestSuite) TestUpsertPodExists() {
	oldPod := fixtures.GetPod()
	oldPod.Instances[0] = &storage.ContainerInstance{
		InstanceId:      expectedPod.Instances[0].InstanceId,
		ContainingPodId: expectedPod.Instances[0].ContainingPodId,
		ContainerName:   expectedPod.Instances[0].ContainerName,
		Started:         expectedPod.Instances[0].Started,
		ImageDigest:     "sha256:092834092384093285190",
		FromRestart:     true,
	}
	oldPod.Instances = append(oldPod.Instances, &storage.ContainerInstance{
		InstanceId: &storage.ContainerInstanceID{
			Id: "newcontainerid",
		},
	})
	merged := proto.Clone(oldPod).(*storage.Pod)
	merged.Instances[0] = expectedPod.Instances[0]
	sort.SliceStable(merged.Instances, func(i, j int) bool { return merged.Instances[i].Started.Compare(merged.Instances[j].Started) <= 0 })
	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(oldPod, true, nil)
	suite.storage.EXPECT().UpsertPod(merged).Return(nil)
	suite.indexer.EXPECT().AddPod(merged).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(merged.GetId()).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsOfStaleContainersByPod(gomock.Any(), merged).Return(nil)
	suite.NoError(suite.datastore.UpsertPod(ctx, expectedPod))
}

func (suite *PodDataStoreTestSuite) TestRemovePod() {
	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().RemovePod(expectedPod.GetId()).Return(nil)
	suite.indexer.EXPECT().DeletePod(expectedPod.GetId()).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(expectedPod.GetId()).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsByPod(gomock.Any(), expectedPod.GetId())
	suite.NoError(suite.datastore.RemovePod(ctx, expectedPod.GetId()))

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, false, nil)
	suite.NoError(suite.datastore.RemovePod(ctx, expectedPod.GetId()))

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, false, errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()))

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().RemovePod(expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().RemovePod(expectedPod.GetId()).Return(nil)
	suite.indexer.EXPECT().DeletePod(expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")

	suite.storage.EXPECT().GetPod(expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().RemovePod(expectedPod.GetId()).Return(nil)
	suite.indexer.EXPECT().DeletePod(expectedPod.GetId()).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")
}

func (suite *PodDataStoreTestSuite) TestReconciliationFullReindex() {
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(true, nil)

	pod1 := fixtures.GetPod()
	pod1.Id = "A"
	pod2 := fixtures.GetPod()
	pod2.Id = "B"

	suite.storage.EXPECT().GetPodIDs().Return([]string{"A", "B", "C"}, nil)
	suite.storage.EXPECT().GetPodsWithIDs([]string{"A", "B", "C"}).Return([]*storage.Pod{pod1, pod2}, nil, nil)
	suite.indexer.EXPECT().AddPods([]*storage.Pod{pod1, pod2}).Return(nil)

	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"D", "E"}, nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"D", "E"}).Return(nil)

	suite.indexer.EXPECT().MarkInitialIndexingComplete().Return(nil)

	// Create a new data store to trigger the reindexing.
	_, err := newDatastoreImpl(suite.storage, suite.indexer, nil, suite.processStore, suite.filter)
	suite.NoError(err)
}

func (suite *PodDataStoreTestSuite) TestReconciliationPartialReindex() {
	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	pod1 := fixtures.GetPod()
	pod1.Id = "A"
	pod2 := fixtures.GetPod()
	pod2.Id = "B"
	pod3 := fixtures.GetPod()
	pod3.Id = "C"

	podList := []*storage.Pod{pod1, pod2, pod3}

	suite.storage.EXPECT().GetPodsWithIDs([]string{"A", "B", "C"}).Return(podList, nil, nil)
	suite.indexer.EXPECT().AddPods(podList).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err := newDatastoreImpl(suite.storage, suite.indexer, nil, suite.processStore, suite.filter)
	suite.NoError(err)

	// Make podList just A,B so C should be deleted
	podList = podList[:1]
	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.storage.EXPECT().GetPodsWithIDs([]string{"A", "B", "C"}).Return(podList, []int{2}, nil)
	suite.indexer.EXPECT().AddPods(podList).Return(nil)
	suite.indexer.EXPECT().DeletePods([]string{"C"}).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err = newDatastoreImpl(suite.storage, suite.indexer, nil, suite.processStore, suite.filter)
	suite.NoError(err)
}
