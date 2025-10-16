package datastore

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	storeMocks "github.com/stackrox/rox/central/pod/datastore/internal/store/mocks"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	plopMocks "github.com/stackrox/rox/central/processlisteningonport/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
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
	processStore *indicatorMocks.MockDataStore
	plopStore    *plopMocks.MockDataStore
	filter       filter.Filter

	mockCtrl *gomock.Controller
}

func (suite *PodDataStoreTestSuite) SetupTest() {
	mockCtrl := gomock.NewController(suite.T())
	suite.mockCtrl = mockCtrl
	suite.storage = storeMocks.NewMockStore(mockCtrl)
	suite.processStore = indicatorMocks.NewMockDataStore(mockCtrl)
	suite.plopStore = plopMocks.NewMockDataStore(mockCtrl)
	suite.filter = filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})

	suite.datastore = newDatastoreImpl(suite.storage, suite.processStore, suite.plopStore, suite.filter)
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

func (suite *PodDataStoreTestSuite) TestGetPod() {
	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	pod, ok, err := suite.datastore.GetPod(ctx, expectedPod.GetId())
	suite.NoError(err)
	suite.True(ok)
	protoassert.Equal(suite.T(), expectedPod, pod)

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
	pod.SetTerminatedInstances(make([]*storage.Pod_ContainerInstanceList, 0))
	// Update one instance.
	ci := &storage.ContainerInstance{}
	ci.SetInstanceId(pod.GetLiveInstances()[0].GetInstanceId())
	ci.SetContainerName(pod.GetLiveInstances()[0].GetContainerName())
	ci.SetImageDigest("sha256:3984274924983274198")
	pod.GetLiveInstances()[0] = ci
	// Terminate the other instance.
	terminatedInst0 := &storage.ContainerInstance{}
	terminatedInst0.SetInstanceId(pod.GetLiveInstances()[1].GetInstanceId())
	terminatedInst0.SetContainerName(pod.GetLiveInstances()[1].GetContainerName())
	terminatedInst0.SetFinished(protocompat.GetProtoTimestampFromSeconds(10))
	terminatedInst0.SetExitCode(0)
	terminatedInst0.SetTerminationReason("Completed")
	pod.GetLiveInstances()[1] = terminatedInst0
	// Add a new terminated instance.
	ciid := &storage.ContainerInstanceID{}
	ciid.SetId("newdeadcontainerid")
	terminatedInst1 := &storage.ContainerInstance{}
	terminatedInst1.SetInstanceId(ciid)
	terminatedInst1.SetContainerName("newdeadcontainername")
	terminatedInst1.SetFinished(protocompat.GetProtoTimestampFromSeconds(9))
	terminatedInst1.SetExitCode(137)
	terminatedInst1.SetTerminationReason("Error")
	pod.SetLiveInstances(append(pod.GetLiveInstances(), terminatedInst1))
	// Add a new live instance.
	ciid2 := &storage.ContainerInstanceID{}
	ciid2.SetId("newlivecontainerid")
	liveInst := &storage.ContainerInstance{}
	liveInst.SetInstanceId(ciid2)
	liveInst.SetContainerName("newlivecontainername")
	liveInst.SetStarted(protocompat.GetProtoTimestampFromSeconds(8))
	pod.SetLiveInstances(append(pod.GetLiveInstances(), liveInst))

	// merged should have all the previously dead instances plus the two new ones
	// as well as the new live instances.
	// This is the pod we expect to actually upsert to the DB.
	merged := fixtures.GetPod()
	merged.SetLiveInstances([]*storage.ContainerInstance{pod.GetLiveInstances()[0], pod.GetLiveInstances()[3]})
	merged.GetTerminatedInstances()[1].SetInstances(append(merged.GetTerminatedInstances()[1].GetInstances(), terminatedInst0))
	pc := &storage.Pod_ContainerInstanceList{}
	pc.SetInstances([]*storage.ContainerInstance{terminatedInst1})
	merged.SetTerminatedInstances(append(merged.GetTerminatedInstances(), pc))
	suite.storage.EXPECT().Get(ctx, pod.GetId()).Return(oldPod, true, nil)
	suite.storage.EXPECT().Upsert(ctx, merged).Return(nil)
	suite.NoError(suite.datastore.UpsertPod(ctx, pod))
}

func (suite *PodDataStoreTestSuite) TestRemovePod() {
	suite.storage.EXPECT().Get(ctx, expectedPod.GetId()).Return(expectedPod, true, nil)
	suite.storage.EXPECT().Delete(ctx, expectedPod.GetId()).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsByPod(gomock.Any(), expectedPod.GetId())
	suite.plopStore.EXPECT().RemovePlopsByPod(gomock.Any(), expectedPod.GetId())
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
