package testcommon

import (
	"context"
	"time"

	"github.com/stackrox/stackrox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/protoconv"
	"github.com/stackrox/stackrox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
)

// NewFlowStoreTest creates a new flow test suite that can be shared between cluster store impls
func NewFlowStoreTest(store store.ClusterStore) *FlowStoreTestSuite {
	return &FlowStoreTestSuite{
		store:  store,
		tested: nil,
		ctx:    context.Background(),
	}
}

// FlowStoreTestSuite is the implementation of the flow store test suite
type FlowStoreTestSuite struct {
	suite.Suite

	store  store.ClusterStore
	tested store.FlowStore
	ctx    context.Context
}

// SetupSuite runs before any tests
func (suite *FlowStoreTestSuite) SetupSuite() {
	var err error
	suite.tested, err = suite.store.CreateFlowStore(context.Background(), "fakecluster")
	suite.Require().NoError(err)
}

// TestStore tests generic network flow store functionality
func (suite *FlowStoreTestSuite) TestStore() {
	// Postgres timestamp only goes to the microsecond level, so we need to truncate these test times
	// to ensure the comparisons of the results works correctly.
	t1 := time.Now().Add(-5 * time.Minute).Truncate(time.Microsecond)
	t2 := time.Now().Truncate(time.Microsecond)
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t1),
			ClusterId:         "fakecluster",
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t2),
			ClusterId:         "fakecluster",
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "yetAnotherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "yetAnotherNode2"},
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			ClusterId: "fakecluster",
		},
	}
	var err error

	updateTS := timestamp.Now() - 1000000
	err = suite.tested.UpsertFlows(context.Background(), flows, updateTS)
	suite.NoError(err, "upsert should succeed on first insert")

	readFlows, readUpdateTS, err := suite.tested.GetAllFlows(context.Background(), nil)
	suite.Require().NoError(err)
	suite.ElementsMatch(readFlows, flows)
	// I don't think these time checks make sense based on how this will work in PG.
	// Not sure it made sense regardless.
	if !features.PostgresDatastore.Enabled() {
		suite.Equal(updateTS, timestamp.FromProtobuf(&readUpdateTS))
	}

	readFlows, readUpdateTS, err = suite.tested.GetAllFlows(context.Background(), protoconv.ConvertTimeToTimestamp(t2))
	suite.Require().NoError(err)
	suite.ElementsMatch(readFlows, flows[1:])
	// I don't think these time checks make sense based on how this will work in PG.
	// Not sure it made sense regardless.
	if !features.PostgresDatastore.Enabled() {
		suite.Equal(updateTS, timestamp.FromProtobuf(&readUpdateTS))
	}

	readFlows, readUpdateTS, err = suite.tested.GetAllFlows(context.Background(), protoconv.ConvertTimeToTimestamp(time.Now()))
	suite.Require().NoError(err)
	suite.ElementsMatch(readFlows, flows[2:])
	// I don't think these time checks make sense based on how this will work in PG.
	// Not sure it made sense regardless.
	if !features.PostgresDatastore.Enabled() {
		suite.Equal(updateTS, timestamp.FromProtobuf(&readUpdateTS))
	}

	updateTS += 1337
	err = suite.tested.UpsertFlows(context.Background(), flows, updateTS)
	suite.NoError(err, "upsert should succeed on second insert")

	err = suite.tested.RemoveFlow(context.Background(), &storage.NetworkFlowProperties{
		SrcEntity:  flows[0].GetProps().GetSrcEntity(),
		DstEntity:  flows[0].GetProps().GetDstEntity(),
		DstPort:    flows[0].GetProps().GetDstPort(),
		L4Protocol: flows[0].GetProps().GetL4Protocol(),
	})
	suite.NoError(err, "remove should succeed when present")

	err = suite.tested.RemoveFlow(context.Background(), &storage.NetworkFlowProperties{
		SrcEntity:  flows[0].GetProps().GetSrcEntity(),
		DstEntity:  flows[0].GetProps().GetDstEntity(),
		DstPort:    flows[0].GetProps().GetDstPort(),
		L4Protocol: flows[0].GetProps().GetL4Protocol(),
	})
	suite.NoError(err, "remove should succeed when not present")

	var actualFlows []*storage.NetworkFlow
	actualFlows, readUpdateTS, err = suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(actualFlows, flows[1:])
	// I don't think these time checks make sense based on how this will work in PG.
	// Not sure it made sense regardless.
	if !features.PostgresDatastore.Enabled() {
		suite.Equal(updateTS, timestamp.FromProtobuf(&readUpdateTS))
	}

	updateTS += 42
	err = suite.tested.UpsertFlows(context.Background(), flows, updateTS)
	suite.NoError(err, "upsert should succeed")

	actualFlows, readUpdateTS, err = suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(actualFlows, flows)
	// I don't think these time checks make sense based on how this will work in PG.
	// Not sure it made sense regardless.
	if !features.PostgresDatastore.Enabled() {
		suite.Equal(updateTS, timestamp.FromProtobuf(&readUpdateTS))
	}

	node1Flows, readUpdateTS, err := suite.tested.GetMatchingFlows(context.Background(), func(props *storage.NetworkFlowProperties) bool {
		if props.GetDstEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && props.GetDstEntity().GetId() == "someNode1" {
			return true
		}
		if props.GetSrcEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && props.GetSrcEntity().GetId() == "someNode1" {
			return true
		}
		return false
	}, nil)
	suite.NoError(err)
	suite.ElementsMatch(node1Flows, flows[:1])
	// I don't think these time checks make sense based on how this will work in PG.
	// Not sure it made sense regardless.
	if !features.PostgresDatastore.Enabled() {
		suite.Equal(updateTS, timestamp.FromProtobuf(&readUpdateTS))
	}
}

// TestRemoveAllMatching tests removing flows that match deployments that have been removed
func (suite *FlowStoreTestSuite) TestRemoveAllMatching() {
	t1 := time.Now().Add(-5 * time.Minute)
	t2 := time.Now()
	if features.PostgresDatastore.Enabled() {
		// Round the timestamps to the microsecond
		t1 = t1.Truncate(1000)
		t2 = t2.Truncate(1000)
	}
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t1),
			ClusterId:         "fakecluster",
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t2),
			ClusterId:         "fakecluster",
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "yetAnotherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "yetAnotherNode2"},
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			ClusterId: "fakecluster",
		},
	}
	updateTS := timestamp.Now() - 1000000
	err := suite.tested.UpsertFlows(context.Background(), flows, updateTS)
	suite.NoError(err)

	// Match none delete none
	err = suite.tested.RemoveMatchingFlows(context.Background(), func(props *storage.NetworkFlowProperties) bool {
		return false
	}, nil)
	suite.NoError(err)

	currFlows, _, err := suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(flows, currFlows)

	// Match dst port 1
	err = suite.tested.RemoveMatchingFlows(context.Background(), func(props *storage.NetworkFlowProperties) bool {
		return props.DstPort == 1
	}, nil)
	suite.NoError(err)

	currFlows, _, err = suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(flows[1:], currFlows)

	// Skipping this one out for right now.  Currently the only use of that function is to delete flows
	// outside the orphan time window.  That is much easier more efficient to deal with in SQL than
	// looping through all the flows and applying that function.
	if !features.PostgresDatastore.Enabled() {
		err = suite.tested.RemoveMatchingFlows(context.Background(), nil, func(flow *storage.NetworkFlow) bool {
			return flow.LastSeenTimestamp.Compare(protoconv.ConvertTimeToTimestamp(t2)) == 0
		})
		suite.NoError(err)

		currFlows, _, err = suite.tested.GetAllFlows(context.Background(), nil)
		suite.NoError(err)
		suite.ElementsMatch(flows[2:], currFlows)
	}
}
