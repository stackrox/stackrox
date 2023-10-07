package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	searcherMocks "github.com/stackrox/rox/central/pod/datastore/internal/search/mocks"
	storeMocks "github.com/stackrox/rox/central/pod/store/mocks"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	plopMocks "github.com/stackrox/rox/central/processlisteningonport/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	searcher     *searcherMocks.MockSearcher
	processStore *indicatorMocks.MockDataStore
	plopStore    *plopMocks.MockDataStore
	filter       filter.Filter

	mockCtrl *gomock.Controller
}

func (suite *PodDataStoreTestSuite) SetupTest() {
	mockCtrl := gomock.NewController(suite.T())
	suite.mockCtrl = mockCtrl
	suite.storage = storeMocks.NewMockStore(mockCtrl)
	suite.searcher = searcherMocks.NewMockSearcher(mockCtrl)
	suite.processStore = indicatorMocks.NewMockDataStore(mockCtrl)
	suite.plopStore = plopMocks.NewMockDataStore(mockCtrl)
	suite.filter = filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})

	suite.datastore = newDatastoreImpl(suite.storage, suite.searcher, suite.processStore, suite.plopStore, suite.filter)
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
	suite.NoError(suite.datastore.UpsertPod(ctx, expectedPod))

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(errors.New("error"))
	suite.Error(suite.datastore.UpsertPod(ctx, expectedPod), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(nil, false, nil)
	suite.storage.EXPECT().Upsert(ctx, expectedPod).Return(nil)
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
	suite.NoError(suite.datastore.UpsertPod(ctx, pod))
}

func (suite *PodDataStoreTestSuite) TestRemovePod() {
	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(nil)
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
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")

	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(errors.New("error"))
	suite.Error(suite.datastore.RemovePod(ctx, expectedPod.GetId()), "error")
}
