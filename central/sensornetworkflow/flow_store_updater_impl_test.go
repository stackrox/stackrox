package sensornetworkflow

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	flowStoreMocks "github.com/stackrox/rox/central/networkflow/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
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
	storedFlows := []*v1.NetworkFlow{
		{
			Props: &v1.NetworkFlowProperties{
				SrcDeploymentId: "someNode1",
				DstDeploymentId: "someNode2",
				DstPort:         1,
				L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: firstTimestamp,
		},
		{
			Props: &v1.NetworkFlowProperties{
				SrcDeploymentId: "someOtherNode1",
				DstDeploymentId: "someOtherNode2",
				DstPort:         2,
				L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
			},
		},
		{
			Props: &v1.NetworkFlowProperties{
				SrcDeploymentId: "someNode1",
				DstDeploymentId: "someOtherNode2",
				DstPort:         2,
				L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: firstTimestamp,
		},
	}

	secondTimestamp := protoconv.ConvertTimeToTimestamp(time.Now())
	newFlows := []*v1.NetworkFlow{
		{
			Props: &v1.NetworkFlowProperties{
				SrcDeploymentId: "someNode1",
				DstDeploymentId: "someNode2",
				DstPort:         1,
				L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: nil,
		},
		{
			Props: &v1.NetworkFlowProperties{
				SrcDeploymentId: "someNode1",
				DstDeploymentId: "someOtherNode2",
				DstPort:         2,
				L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: secondTimestamp,
		},
	}

	// The properties of the flows we expect updates to. Properties identify flows uniquely.
	expectedUpdateProps := []*v1.NetworkFlowProperties{
		{
			SrcDeploymentId: "someNode1",
			DstDeploymentId: "someNode2",
			DstPort:         1,
			L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
		},
		{
			SrcDeploymentId: "someOtherNode1",
			DstDeploymentId: "someOtherNode2",
			DstPort:         2,
			L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
		},
		{
			SrcDeploymentId: "someNode1",
			DstDeploymentId: "someOtherNode2",
			DstPort:         2,
			L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	// Return storedFlows on DB read.
	suite.mockFlowStore.EXPECT().GetAllFlows().Return(storedFlows, *firstTimestamp, nil)

	// Check that the given write matches expectations.
	suite.mockFlowStore.EXPECT().UpsertFlows(testutils.PredMatcher("matches expected updates", func(actualUpdates []*v1.NetworkFlow) bool {
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
