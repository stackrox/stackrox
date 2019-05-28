package networkflowupdate

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	nfDSMocks "github.com/stackrox/rox/central/networkflow/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestFlowStoreUpdater(t *testing.T) {
	suite.Run(t, new(FlowStoreUpdaterTestSuite))
}

type FlowStoreUpdaterTestSuite struct {
	suite.Suite

	mockFlows *nfDSMocks.MockFlowDataStore
	tested    flowStoreUpdater

	mockCtrl    *gomock.Controller
	hasReadCtx  context.Context
	hasWriteCtx context.Context
}

func (suite *FlowStoreUpdaterTestSuite) SetupSuite() {
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
	suite.tested = newFlowStoreUpdater(suite.mockFlows)
}

func (suite *FlowStoreUpdaterTestSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
}

func (suite *FlowStoreUpdaterTestSuite) TestUpdate() {
	firstTimestamp := protoconv.ConvertTimeToTimestamp(time.Now())
	storedFlows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: firstTimestamp,
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
			LastSeenTimestamp: firstTimestamp,
		},
	}

	secondTimestamp := protoconv.ConvertTimeToTimestamp(time.Now())
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
			LastSeenTimestamp: secondTimestamp,
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
	}

	// Return storedFlows on DB read.
	suite.mockFlows.EXPECT().GetAllFlows(suite.hasWriteCtx, gomock.Any()).Return(storedFlows, *firstTimestamp, nil)

	// Check that the given write matches expectations.
	suite.mockFlows.EXPECT().UpsertFlows(suite.hasWriteCtx, testutils.PredMatcher("matches expected updates", func(actualUpdates []*storage.NetworkFlow) bool {
		if len(actualUpdates) != len(expectedUpdateProps) {
			return false
		}
		used := make(map[int]bool)
		for _, actualUpdate := range actualUpdates {
			for index, expectedProp := range expectedUpdateProps {
				if proto.Equal(actualUpdate.GetProps(), expectedProp) {
					if used[index] {
						return false
					}
					used[index] = true
				}
			}
		}
		return len(used) == len(expectedUpdateProps)
	}), gomock.Any()).Return(nil)

	// Run test.
	err := suite.tested.update(suite.hasWriteCtx, newFlows, secondTimestamp)
	suite.NoError(err, "update should succeed on first insert")
}
