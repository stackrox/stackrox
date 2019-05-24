package store

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/mocks"
	undoStoreMocks "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
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

	dataStore   DataStore
	storage     *storeMocks.MockStore
	undoStorage *undoStoreMocks.MockUndoStore
	mockCtrl    *gomock.Controller
}

func (s *netPolDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasNS1ReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy), sac.NamespaceScopeKeys(FakeNamespace1)))
	s.hasNS2ReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy), sac.NamespaceScopeKeys(FakeNamespace2)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.undoStorage = undoStoreMocks.NewMockUndoStore(s.mockCtrl)
	s.dataStore = New(s.storage, s.undoStorage)
}

func (s *netPolDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *netPolDataStoreTestSuite) TestEnforceGet() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.storage.EXPECT().GetNetworkPolicy(gomock.Any()).Times(0)

	netPol, found, err := s.dataStore.GetNetworkPolicy(s.hasNoneCtx, FakeID1)
	s.Error(err, "expected an error trying to write without permissions")
	s.False(found)
	s.Nil(netPol, "expected return value to be nil")
}

func (s *netPolDataStoreTestSuite) TestEnforceGetAll() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.storage.EXPECT().GetNetworkPolicy(gomock.Any()).Times(0)

	netPol, found, err := s.dataStore.GetNetworkPolicy(s.hasNoneCtx, FakeID1)
	s.NoError(err, "expected no error, should return nil without access")
	s.False(found)
	s.Nil(netPol, "expected return value to be nil")
}

func (s *netPolDataStoreTestSuite) TestEnforcesAdd() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.storage.EXPECT().AddNetworkPolicy(gomock.Any()).Times(0)

	err := s.dataStore.AddNetworkPolicy(s.hasNoneCtx, &storage.NetworkPolicy{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddNetworkPolicy(s.hasNS1ReadCtx, &storage.NetworkPolicy{})
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

	filterResult := []*storage.NetworkPolicy{
		netPolNm1,
	}
	s.storage.EXPECT().AddNetworkPolicy(gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.AddNetworkPolicy(s.hasNS1ReadCtx, netPolNm1)
	s.NoError(err)
	err = s.dataStore.AddNetworkPolicy(s.hasNS2ReadCtx, netPolNm2)
	s.NoError(err)

	s.storage.EXPECT().GetNetworkPolicy(FakeID1).Return(netPolNm1, true, nil)

	result, found, err := s.dataStore.GetNetworkPolicy(s.hasNS1ReadCtx, FakeID1)
	s.NoError(err)
	s.True(found)
	s.Equal(result, netPolNm1)

	s.storage.EXPECT().GetNetworkPolicy(FakeID2).Return(netPolNm2, true, nil)

	result, found, err = s.dataStore.GetNetworkPolicy(s.hasNS2ReadCtx, FakeID2)
	s.NoError(err)
	s.True(found)
	s.Equal(result, netPolNm2)

	s.storage.EXPECT().GetNetworkPolicies(FakeClusterID, gomock.Any()).Return(filterResult, nil).Times(2)

	netPols, err := s.dataStore.GetNetworkPolicies(s.hasNS1ReadCtx, FakeClusterID, FakeNamespace1)
	s.NoError(err)
	s.Equal(netPols, filterResult)

	netPols, err = s.dataStore.GetNetworkPolicies(s.hasNS1ReadCtx, FakeClusterID, "")
	s.NoError(err)
	s.Equal(netPols, filterResult)
}

func (s *netPolDataStoreTestSuite) TestEnforcesUpdate() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.storage.EXPECT().UpdateNetworkPolicy(gomock.Any()).Times(0)

	err := s.dataStore.UpdateNetworkPolicy(s.hasNoneCtx, &storage.NetworkPolicy{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateNetworkPolicy(s.hasNS1ReadCtx, &storage.NetworkPolicy{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestAllowsUpdate() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.storage.EXPECT().UpdateNetworkPolicy(gomock.Any()).Times(0)

	err := s.dataStore.UpdateNetworkPolicy(s.hasWriteCtx, &storage.NetworkPolicy{})
	s.NoError(err, "expected no error, should return nil without access")
}

func (s *netPolDataStoreTestSuite) TestEnforcesRemove() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.storage.EXPECT().GetNetworkPolicy(gomock.Any()).Times(0)
	s.storage.EXPECT().RemoveNetworkPolicy(gomock.Any()).Times(0)

	err := s.dataStore.RemoveNetworkPolicy(s.hasNoneCtx, FakeID1)
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveNetworkPolicy(s.hasNS1ReadCtx, FakeID1)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestAllowsRemove() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.storage.EXPECT().GetNetworkPolicy(gomock.Any()).Times(0)
	s.storage.EXPECT().RemoveNetworkPolicy(gomock.Any()).Times(0)

	err := s.dataStore.RemoveNetworkPolicy(s.hasWriteCtx, FakeID1)
	s.NoError(err, "expected no error, should return nil without access")
}

func (s *netPolDataStoreTestSuite) TestEnforceGetUndo() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.undoStorage.EXPECT().GetUndoRecord(gomock.Any()).Times(0)

	_, found, err := s.dataStore.GetUndoRecord(s.hasNoneCtx, FakeID1)
	s.Error(err, "expected an error trying to write without permissions")
	s.False(found)

	_, found, err = s.dataStore.GetUndoRecord(s.hasNS1ReadCtx, FakeID1)
	s.Error(err, "expected an error trying to write without permissions")
	s.False(found)
}

func (s *netPolDataStoreTestSuite) TestAllowGetUndo() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.undoStorage.EXPECT().GetUndoRecord(gomock.Any()).Times(0)

	_, found, err := s.dataStore.GetUndoRecord(s.hasWriteCtx, FakeClusterID)
	s.NoError(err, "expected an error trying to write without permissions")
	s.True(found)
}

func (s *netPolDataStoreTestSuite) TestEnforceUpdateUndo() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.undoStorage.EXPECT().UpsertUndoRecord(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpsertUndoRecord(s.hasNoneCtx, FakeClusterID, &storage.NetworkPolicyApplicationUndoRecord{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertUndoRecord(s.hasNS1ReadCtx, FakeClusterID, &storage.NetworkPolicyApplicationUndoRecord{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *netPolDataStoreTestSuite) TestAllowUpdateUndo() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().Skip()
	}
	s.undoStorage.EXPECT().UpsertUndoRecord(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpsertUndoRecord(s.hasWriteCtx, FakeClusterID, &storage.NetworkPolicyApplicationUndoRecord{})
	s.NoError(err, "expected an error trying to write without permissions")
}
