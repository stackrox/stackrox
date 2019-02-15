package networkflowupdate

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	flowStoreMocks "github.com/stackrox/rox/central/networkflow/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestFlowStoreUpdater(t *testing.T) {
	suite.Run(t, new(FlowStoreUpdaterTestSuite))
}

type FlowStoreUpdaterTestSuite struct {
	suite.Suite

	mockFlowStore *flowStoreMocks.MockFlowStore
	tested        flowStoreUpdater

	mockCtrl *gomock.Controller
}

func (suite *FlowStoreUpdaterTestSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockFlowStore = flowStoreMocks.NewMockFlowStore(suite.mockCtrl)
	suite.tested = newFlowStoreUpdater(suite.mockFlowStore)
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
	suite.mockFlowStore.EXPECT().GetAllFlows(gomock.Any()).Return(storedFlows, *firstTimestamp, nil)

	// Check that the given write matches expectations.
	suite.mockFlowStore.EXPECT().UpsertFlows(testutils.PredMatcher("matches expected updates", func(actualUpdates []*storage.NetworkFlow) bool {
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
	var err error
	err = suite.tested.update(newFlows, secondTimestamp)
	suite.NoError(err, "update should succeed on first insert")
}
