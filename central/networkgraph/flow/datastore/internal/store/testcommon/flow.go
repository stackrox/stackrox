//go:build sql_integration

package testcommon

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
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
	suite.tested, err = suite.store.CreateFlowStore(context.Background(), fixtureconsts.Cluster1)
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
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment1},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment2},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t1),
			ClusterId:         fixtureconsts.Cluster1,
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment3},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment4},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t2),
			ClusterId:         fixtureconsts.Cluster1,
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment5},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment6},
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			ClusterId: fixtureconsts.Cluster1,
		},
	}
	var err error

	updateTS := timestamp.Now() - 1000000
	err = suite.tested.UpsertFlows(context.Background(), flows, updateTS)
	suite.NoError(err, "upsert should succeed on first insert")

	readFlows, _, err := suite.tested.GetAllFlows(context.Background(), nil)
	suite.Require().NoError(err)
	suite.ElementsMatch(readFlows, flows)

	readFlows, _, err = suite.tested.GetAllFlows(context.Background(), protoconv.ConvertTimeToTimestamp(t2))
	suite.Require().NoError(err)
	suite.ElementsMatch(readFlows, flows[1:])

	readFlows, _, err = suite.tested.GetAllFlows(context.Background(), protoconv.ConvertTimeToTimestamp(time.Now()))
	suite.Require().NoError(err)
	suite.ElementsMatch(readFlows, flows[2:])

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
	actualFlows, _, err = suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(actualFlows, flows[1:])

	updateTS += 42
	err = suite.tested.UpsertFlows(context.Background(), flows, updateTS)
	suite.NoError(err, "upsert should succeed")

	actualFlows, _, err = suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(actualFlows, flows)

	node1Flows, _, err := suite.tested.GetMatchingFlows(context.Background(), func(props *storage.NetworkFlowProperties) bool {
		if props.GetDstEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && props.GetDstEntity().GetId() == fixtureconsts.Deployment1 {
			return true
		}
		if props.GetSrcEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && props.GetSrcEntity().GetId() == fixtureconsts.Deployment1 {
			return true
		}
		return false
	}, nil)
	suite.NoError(err)
	suite.ElementsMatch(node1Flows, flows[:1])
}

// TestRemoveAllMatching tests removing flows that match deployments that have been removed
func (suite *FlowStoreTestSuite) TestRemoveAllMatching() {
	t1 := time.Now().Add(-5 * time.Minute)
	t2 := time.Now()
	t3 := time.Now().Add(15 * time.Minute)
	// Round the timestamps to the microsecond
	t1 = t1.Truncate(1000)
	t2 = t2.Truncate(1000)
	t3 = t3.Truncate(1000)
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment1},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment2},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t1),
			ClusterId:         fixtureconsts.Cluster1,
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment3},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment4},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t2),
			ClusterId:         fixtureconsts.Cluster1,
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment5},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment6},
				DstPort:    3,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			ClusterId: fixtureconsts.Cluster1,
		},
	}
	updateTS := timestamp.Now() - 1000000
	err := suite.tested.UpsertFlows(context.Background(), flows, updateTS)
	suite.NoError(err)

	currFlows, _, err := suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(flows, currFlows)

	utc := t3.UTC()
	err = suite.tested.RemoveOrphanedFlows(context.Background(), &utc)
	suite.NoError(err)

	currFlows, _, err = suite.tested.GetAllFlows(context.Background(), nil)
	suite.NoError(err)
	suite.ElementsMatch(flows[2:], currFlows)
}
