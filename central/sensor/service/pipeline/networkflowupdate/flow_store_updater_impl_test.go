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
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"}.Build(),
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestampOrNow(&firstTimestamp),
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"}.Build(),
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"}.Build(),
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"}.Build(),
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestampOrNow(&firstTimestamp),
		}.Build(),
	}

	discoveredEntity1 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{1, 2, 3, 4, 32})).ToProto()
	discoveredEntity2 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{2, 3, 4, 5, 32})).ToProto()

	secondTimestamp := time.Now()
	newFlows := []*storage.NetworkFlow{
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"}.Build(),
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: nil,
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"}.Build(),
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstEntity:  discoveredEntity1,
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  discoveredEntity2,
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstPort:    4,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
	}

	// The properties of the flows we expect updates to. Properties identify flows uniquely.
	expectedUpdateProps := []*storage.NetworkFlowProperties{
		storage.NetworkFlowProperties_builder{
			SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
			DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"}.Build(),
			DstPort:    1,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
		storage.NetworkFlowProperties_builder{
			SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"}.Build(),
			DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"}.Build(),
			DstPort:    2,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
		storage.NetworkFlowProperties_builder{
			SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
			DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"}.Build(),
			DstPort:    2,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
		storage.NetworkFlowProperties_builder{
			SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
			DstEntity:  networkgraph.InternetEntity().ToProto(), // features.ExternalIPs is disabled
			DstPort:    3,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
		storage.NetworkFlowProperties_builder{
			SrcEntity:  networkgraph.InternetEntity().ToProto(), // features.ExternalIPs is disabled
			DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
			DstPort:    4,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
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
	}), gomock.Any()).Return(newFlows, nil)

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
	discoveredEntity3 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{3, 4, 5, 6, 32})).ToProto()
	fixedupDiscoveredEntity1 := networkgraph.DiscoveredExternalEntityClusterScoped("cluster", net.IPNetworkFromCIDRBytes([]byte{1, 2, 3, 4, 32})).ToProto()
	fixedupDiscoveredEntity2 := networkgraph.DiscoveredExternalEntityClusterScoped("cluster", net.IPNetworkFromCIDRBytes([]byte{2, 3, 4, 5, 32})).ToProto()
	fixedupDiscoveredEntity3 := networkgraph.DiscoveredExternalEntityClusterScoped("cluster", net.IPNetworkFromCIDRBytes([]byte{3, 4, 5, 6, 32})).ToProto()

	secondTimestamp := time.Now()
	newFlows := []*storage.NetworkFlow{
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstEntity:  discoveredEntity1,
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  discoveredEntity2,
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstPort:    4,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  discoveredEntity3,
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstPort:    4,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
	}

	actuallyUpsertedFlows := []*storage.NetworkFlow{
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstEntity:  discoveredEntity1,
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
		storage.NetworkFlow_builder{
			Props: storage.NetworkFlowProperties_builder{
				SrcEntity:  discoveredEntity2,
				DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
				DstPort:    4,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}.Build(),
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(secondTimestamp),
		}.Build(),
		// We simulate that the third flow was filtered out during Upsert().
	}

	// The properties of the flows we expect updates to. Properties identify flows uniquely.
	expectedUpdateProps := []*storage.NetworkFlowProperties{
		storage.NetworkFlowProperties_builder{
			SrcEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
			DstEntity:  fixedupDiscoveredEntity1,
			DstPort:    3,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
		storage.NetworkFlowProperties_builder{
			SrcEntity:  fixedupDiscoveredEntity2,
			DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
			DstPort:    4,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
		storage.NetworkFlowProperties_builder{
			SrcEntity:  fixedupDiscoveredEntity3,
			DstEntity:  storage.NetworkEntityInfo_builder{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"}.Build(),
			DstPort:    4,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
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
					ID:                    "cluster__MS4yLjMuNC8zMg",
					ExternalEntityAddress: net.IPNetworkFromCIDRBytes([]byte{1, 2, 3, 4, 32}),
					Discovered:            true,
				},
				DstPort:  3,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			}: timestamp.FromGoTime(secondTimestamp),
			{
				SrcEntity: networkgraph.Entity{
					Type:                  storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ID:                    "cluster__Mi4zLjQuNS8zMg",
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
			{
				SrcEntity: networkgraph.Entity{
					Type:                  storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ID:                    "cluster__My40LjUuNi8zMg",
					ExternalEntityAddress: net.IPNetworkFromCIDRBytes([]byte{3, 4, 5, 6, 32}),
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
	}), gomock.Any()).Return(actuallyUpsertedFlows, nil)

	suite.mockEntities.EXPECT().UpdateExternalNetworkEntity(suite.hasWriteCtx, testutils.PredMatcher("matches an external entity", func(updatedEntity *storage.NetworkEntity) bool {
		expectedEntities := []*storage.NetworkEntityInfo{
			discoveredEntity1,
			discoveredEntity2,
			// not discoveredEntity3 since the flow was filtered
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
