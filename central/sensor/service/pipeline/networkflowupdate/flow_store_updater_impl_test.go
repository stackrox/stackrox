package networkflowupdate

import (
	"context"
	"math"
	"testing"
	"time"

	baselineMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	entityMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	nfDSMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestFlowStoreUpdater(t *testing.T) {
	suite.Run(t, new(FlowStoreUpdaterTestSuite))
}

type FlowStoreUpdaterTestSuite struct {
	suite.Suite

	mockFlows     *nfDSMocks.MockFlowDataStore
	mockBaselines *baselineMocks.MockManager
	mockEntities  *entityMocks.MockEntityDataStore
	tested        flowPersister

	mockCtrl    *gomock.Controller
	hasReadCtx  context.Context
	hasWriteCtx context.Context
}

func (suite *FlowStoreUpdaterTestSuite) SetupTest() {
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy, resources.NetworkGraph)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy, resources.NetworkGraph)))

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockFlows = nfDSMocks.NewMockFlowDataStore(suite.mockCtrl)
	suite.mockBaselines = baselineMocks.NewMockManager(suite.mockCtrl)
	suite.mockEntities = entityMocks.NewMockEntityDataStore(suite.mockCtrl)
	suite.tested = newFlowPersister(suite.mockFlows, suite.mockBaselines, suite.mockEntities, "cluster")
}

func (suite *FlowStoreUpdaterTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *FlowStoreUpdaterTestSuite) TestUpdateNoExternalIPs() {
	suite.T().Setenv(features.ExternalIPs.EnvVar(), "false")

	firstTimestamp := time.Now()
	storedFlows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestampOrNow(&firstTimestamp),
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestampOrNow(&firstTimestamp),
		},
	}

	discoveredEntity1 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{1, 2, 3, 4, 32})).ToProto()
	discoveredEntity2 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{2, 3, 4, 5, 32})).ToProto()

	secondTimestamp := time.Now()
	newFlows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: nil,
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  discoveredEntity1,
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  discoveredEntity2,
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstPort:    4,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		},
	}

	// The properties of the flows we expect updates to. Properties identify flows uniquely.
	expectedUpdateProps := []*storage.NetworkFlowProperties{
		{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
			DstPort:    1,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"},
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
			DstPort:    2,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
			DstPort:    2,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
			DstEntity:  networkgraph.InternetEntity().ToProto(), // features.ExternalIPs is disabled
			DstPort:    3,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		{
			SrcEntity:  networkgraph.InternetEntity().ToProto(), // features.ExternalIPs is disabled
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
			DstPort:    4,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	// Return storedFlows on DB read.
	suite.mockFlows.EXPECT().GetAllFlows(suite.hasWriteCtx, gomock.Any()).Return(storedFlows, &firstTimestamp, nil)

	suite.mockBaselines.EXPECT().ProcessFlowUpdate(testutils.PredMatcher("equivalent map except for timestamp", func(got map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) bool {
		expectedMap := map[networkgraph.NetworkConnIndicator]timestamp.MicroTS{
			{
				SrcEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someNode1",
				},
				DstEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someNode2",
				},
				DstPort:  1,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}: 0,
			{
				SrcEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someNode1",
				},
				DstEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someOtherNode2",
				},
				DstPort:  2,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}: timestamp.FromGoTime(secondTimestamp),
			{
				SrcEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someNode1",
				},
				DstEntity: networkgraph.InternetEntity(), // features.ExternalIPs is disabled
				DstPort:   3,
				Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
			}: timestamp.FromGoTime(secondTimestamp),
			{
				SrcEntity: networkgraph.InternetEntity(), // features.ExternalIPs is disabled
				DstEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someNode1",
				},
				DstPort:  4,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}: timestamp.FromGoTime(secondTimestamp),
		}

		if len(expectedMap) != len(got) {
			return false
		}
		for indicator, ts := range expectedMap {
			got, inGot := got[indicator]
			if !inGot {
				return false
			}
			if ts == 0 {
				if got != 0 {
					return false
				}
			} else {
				// The timestamp may vary slightly because of the adjustment that we do,
				// but should not vary by more than a second.
				if math.Abs(ts.GoTime().Sub(got.GoTime()).Seconds()) > 1 {
					return false
				}
			}
		}
		return true
	},
	))

	// Check that the given write matches expectations.
	suite.mockFlows.EXPECT().UpsertFlows(suite.hasWriteCtx, testutils.PredMatcher("matches expected updates", func(actualUpdates []*storage.NetworkFlow) bool {
		if len(actualUpdates) != len(expectedUpdateProps) {
			return false
		}
		used := make(map[int]bool)
		for _, actualUpdate := range actualUpdates {
			for index, expectedProp := range expectedUpdateProps {
				if actualUpdate.GetProps().EqualVT(expectedProp) {
					if used[index] {
						return false
					}
					used[index] = true
				}
			}
		}
		return len(used) == len(expectedUpdateProps)
	}), gomock.Any()).Return(nil)

	suite.mockEntities.EXPECT().UpdateExternalNetworkEntity(suite.hasWriteCtx, gomock.Any(), true).Times(0)

	// Run test.
	err := suite.tested.update(suite.hasWriteCtx, newFlows, &secondTimestamp)
	suite.NoError(err, "update should succeed on first insert")
}

func (suite *FlowStoreUpdaterTestSuite) TestUpdateWithExternalIPs() {
	suite.T().Setenv(features.ExternalIPs.EnvVar(), "true")

	firstTimestamp := time.Now()
	storedFlows := []*storage.NetworkFlow{}

	discoveredEntity1 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{1, 2, 3, 4, 32})).ToProto()
	discoveredEntity2 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{2, 3, 4, 5, 32})).ToProto()

	secondTimestamp := time.Now()
	newFlows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  discoveredEntity1,
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  discoveredEntity2,
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstPort:    4,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		},
	}

	// The properties of the flows we expect updates to. Properties identify flows uniquely.
	expectedUpdateProps := []*storage.NetworkFlowProperties{
		{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
			DstEntity:  discoveredEntity1,
			DstPort:    3,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		{
			SrcEntity:  discoveredEntity2,
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
			DstPort:    4,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	// Return storedFlows on DB read.
	suite.mockFlows.EXPECT().GetAllFlows(suite.hasWriteCtx, gomock.Any()).Return(storedFlows, &firstTimestamp, nil)

	suite.mockBaselines.EXPECT().ProcessFlowUpdate(testutils.PredMatcher("equivalent map except for timestamp", func(got map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) bool {
		expectedMap := map[networkgraph.NetworkConnIndicator]timestamp.MicroTS{
			{
				SrcEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someNode1",
				},
				DstEntity: networkgraph.Entity{
					Type:                  storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ID:                    "__MS4yLjMuNC8zMg",
					ExternalEntityAddress: net.IPNetworkFromCIDRBytes([]byte{1, 2, 3, 4, 32}),
					Discovered:            true,
				},
				DstPort:  3,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}: timestamp.FromGoTime(secondTimestamp),
			{
				SrcEntity: networkgraph.Entity{
					Type:                  storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ID:                    "__Mi4zLjQuNS8zMg",
					ExternalEntityAddress: net.IPNetworkFromCIDRBytes([]byte{2, 3, 4, 5, 32}),
					Discovered:            true,
				},
				DstEntity: networkgraph.Entity{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					ID:   "someNode1",
				},
				DstPort:  4,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}: timestamp.FromGoTime(secondTimestamp),
		}

		if len(expectedMap) != len(got) {
			return false
		}
		for indicator, ts := range expectedMap {
			got, inGot := got[indicator]
			if !inGot {
				return false
			}
			if ts == 0 {
				if got != 0 {
					return false
				}
			} else {
				// The timestamp may vary slightly because of the adjustment that we do,
				// but should not vary by more than a second.
				if math.Abs(ts.GoTime().Sub(got.GoTime()).Seconds()) > 1 {
					return false
				}
			}
		}
		return true
	},
	))

	// Check that the given write matches expectations.
	suite.mockFlows.EXPECT().UpsertFlows(suite.hasWriteCtx, testutils.PredMatcher("matches expected updates", func(actualUpdates []*storage.NetworkFlow) bool {
		if len(actualUpdates) != len(expectedUpdateProps) {
			return false
		}
		used := make(map[int]bool)
		for _, actualUpdate := range actualUpdates {
			for index, expectedProp := range expectedUpdateProps {
				if actualUpdate.GetProps().EqualVT(expectedProp) {
					if used[index] {
						return false
					}
					used[index] = true
				}
			}
		}
		return len(used) == len(expectedUpdateProps)
	}), gomock.Any()).Return(nil)

	suite.mockEntities.EXPECT().UpdateExternalNetworkEntity(suite.hasWriteCtx, testutils.PredMatcher("matches an external entity", func(updatedEntity *storage.NetworkEntity) bool {
		expectedEntities := []*storage.NetworkEntityInfo{
			discoveredEntity1,
			discoveredEntity2,
		}

		for i := range expectedEntities {
			if updatedEntity.GetInfo().EqualVT(expectedEntities[i]) {
				return true
			}
		}
		return false
	}), true).Times(2).Return(nil)

	// Run test.
	err := suite.tested.update(suite.hasWriteCtx, newFlows, &secondTimestamp)
	suite.NoError(err, "update should succeed on first insert")
}
