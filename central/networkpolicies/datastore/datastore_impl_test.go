package store

import (
	"context"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	storeMocks "github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/mocks"
	undoDeploymentStoreMocks "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/mocks"
	undoStoreMocks "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNetPolDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(netPolDataStoreTestSuite))
}

const (
	FakeID1        = "FAKEID_1"
	FakeID2        = "FAKEID_2"
	FakeName1      = "FAKENAME_1"
	FakeName2      = "FAKENAME_2"
	FakeClusterID  = "CLUSTER_1"
	FakeNamespace1 = "NAMESPACE_1"
	FakeNamespace2 = "NAMESPACE_2"
)

type netPolDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx    context.Context
	hasNS1ReadCtx context.Context
	hasNS2ReadCtx context.Context
	hasWriteCtx   context.Context

	dataStore             DataStore
	storage               *storeMocks.MockStore
	undoStorage           *undoStoreMocks.MockUndoStore
	undoDeploymentStorage *undoDeploymentStoreMocks.MockUndoDeploymentStore
	mockCtrl              *gomock.Controller
}

func (s *netPolDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasNS1ReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedNamespaceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy),
			sac.ClusterScopeKeys(FakeClusterID),
			sac.NamespaceScopeKeys(FakeNamespace1)))
	s.hasNS2ReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedNamespaceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy),
			sac.ClusterScopeKeys(FakeClusterID),
			sac.NamespaceScopeKeys(FakeNamespace2)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.undoStorage = undoStoreMocks.NewMockUndoStore(s.mockCtrl)
	s.undoDeploymentStorage = undoDeploymentStoreMocks.NewMockUndoDeploymentStore(s.mockCtrl)
	s.dataStore = New(s.storage, nil, s.undoStorage, s.undoDeploymentStorage)
}

func (s *netPolDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *netPolDataStoreTestSuite) TestEnforceGet() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.NetworkPolicy{}, true, nil)

	netPol, found, err := s.dataStore.GetNetworkPolicy(s.hasNoneCtx, FakeID1)
	s.NoError(err, "expected an error trying to write without permissions")
	s.False(found)
	s.Nil(netPol, "expected return value to be nil")
}

func (s *netPolDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpsertNetworkPolicy(s.hasNoneCtx, &storage.NetworkPolicy{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertNetworkPolicy(s.hasNS1ReadCtx, &storage.NetworkPolicy{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestGetNetworkPolicies() {
	netPolNm1 := &storage.NetworkPolicy{
		Id:        FakeID1,
		Name:      FakeName1,
		ClusterId: FakeClusterID,
		Namespace: FakeNamespace1,
	}
	netPolNm2 := &storage.NetworkPolicy{
		Id:        FakeID2,
		Name:      FakeName2,
		ClusterId: FakeClusterID,
		Namespace: FakeNamespace2,
	}

	// Test we can get with NS1 permissions
	s.storage.EXPECT().Get(gomock.Any(), FakeID1).Return(netPolNm1, true, nil)

	result, found, err := s.dataStore.GetNetworkPolicy(s.hasNS1ReadCtx, FakeID1)
	s.NoError(err)
	s.True(found)
	s.Equal(result, netPolNm1)

	// Test we can get with NS2 permissions.
	s.storage.EXPECT().Get(gomock.Any(), FakeID2).Return(netPolNm2, true, nil)

	result, found, err = s.dataStore.GetNetworkPolicy(s.hasNS2ReadCtx, FakeID2)
	s.NoError(err)
	s.True(found)
	s.Equal(result, netPolNm2)

	// Test we cannot do the opposite.
	s.storage.EXPECT().GetByQuery(gomock.Any(), gomock.Any()).Return(nil, nil)

	netPols, err := s.dataStore.GetNetworkPolicies(s.hasNS1ReadCtx, FakeClusterID, FakeNamespace2)
	s.NoError(err)
	s.Equal(0, len(netPols))

	s.storage.EXPECT().GetByQuery(gomock.Any(), gomock.Any()).Return(nil, nil)

	netPols, err = s.dataStore.GetNetworkPolicies(s.hasNS2ReadCtx, FakeClusterID, FakeNamespace1)
	s.NoError(err)
	s.Equal(0, len(netPols))
}

func (s *netPolDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpsertNetworkPolicy(s.hasNoneCtx, &storage.NetworkPolicy{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertNetworkPolicy(s.hasNS1ReadCtx, &storage.NetworkPolicy{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	err := s.dataStore.UpsertNetworkPolicy(s.hasWriteCtx, &storage.NetworkPolicy{})
	s.NoError(err, "expected no error, should return nil without access")
}

func (s *netPolDataStoreTestSuite) TestEnforcesRemove() {
	// None should be removed...
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	// ...whether we have no access...
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.NetworkPolicy{}, true, nil)

	err := s.dataStore.RemoveNetworkPolicy(s.hasNoneCtx, FakeID1)
	s.Error(err, "expected an error trying to write without permissions")

	// ...or we only have read access.
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.NetworkPolicy{}, true, nil)

	err = s.dataStore.RemoveNetworkPolicy(s.hasNS1ReadCtx, FakeID1)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.NetworkPolicy{}, true, nil)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

	err := s.dataStore.RemoveNetworkPolicy(s.hasWriteCtx, FakeID1)
	s.NoError(err, "expected no error, should return nil without access")
}

func (s *netPolDataStoreTestSuite) TestEnforceGetUndo() {
	s.undoStorage.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)

	_, found, err := s.dataStore.GetUndoRecord(s.hasNoneCtx, FakeID1)
	s.NoError(err, "expected no error trying to read without permissions")
	s.False(found)

	_, found, err = s.dataStore.GetUndoRecord(s.hasNS1ReadCtx, FakeID1)
	s.NoError(err, "expected no error trying to read without permissions")
	s.False(found)
}

func (s *netPolDataStoreTestSuite) TestAllowGetUndo() {
	s.undoStorage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.NetworkPolicyApplicationUndoRecord{}, true, nil)

	_, found, err := s.dataStore.GetUndoRecord(s.hasWriteCtx, FakeClusterID)
	s.NoError(err, "expected an error trying to write without permissions")
	s.True(found)
}

func (s *netPolDataStoreTestSuite) TestEnforceUpdateUndo() {
	s.undoStorage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpsertUndoRecord(s.hasNoneCtx, &storage.NetworkPolicyApplicationUndoRecord{ClusterId: FakeClusterID})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertUndoRecord(s.hasNS1ReadCtx, &storage.NetworkPolicyApplicationUndoRecord{ClusterId: FakeClusterID})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestAllowUpdateUndo() {
	s.undoStorage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil)
	s.undoStorage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	err := s.dataStore.UpsertUndoRecord(s.hasWriteCtx, &storage.NetworkPolicyApplicationUndoRecord{ClusterId: FakeClusterID})
	s.NoError(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestAllowUpdateUndoNewer() {
	oldCluster := &storage.NetworkPolicyApplicationUndoRecord{ClusterId: FakeClusterID, ApplyTimestamp: timestamp.TimestampNow()}
	s.undoStorage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(oldCluster, true, nil)
	s.undoStorage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	err := s.dataStore.UpsertUndoRecord(s.hasWriteCtx, &storage.NetworkPolicyApplicationUndoRecord{ClusterId: FakeClusterID, ApplyTimestamp: timestamp.TimestampNow()})
	s.NoError(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestDisallowUpdateUndoOlder() {
	oldTS := timestamp.TimestampNow()
	newTS := timestamp.TimestampNow()
	// Ensure the timestamps differ
	newTS.Nanos += 1000
	oldCluster := &storage.NetworkPolicyApplicationUndoRecord{ClusterId: FakeClusterID, ApplyTimestamp: newTS}
	s.undoStorage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(oldCluster, true, nil)

	err := s.dataStore.UpsertUndoRecord(s.hasWriteCtx, &storage.NetworkPolicyApplicationUndoRecord{ClusterId: FakeClusterID, ApplyTimestamp: oldTS})
	s.Error(err, "expected an error trying to write without permissions")
}
