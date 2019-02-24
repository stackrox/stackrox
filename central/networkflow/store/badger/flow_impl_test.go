package badger

import (
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	suite.Run(t, new(FlowStoreTestSuite))
}

type FlowStoreTestSuite struct {
	suite.Suite

	path   string
	db     *badger.DB
	tested store.FlowStore
}

func (suite *FlowStoreTestSuite) SetupSuite() {
	db, path, err := badgerhelper.NewTemp(suite.T().Name())
	suite.Require().NoError(err)
	suite.db = db
	suite.path = path

	cs := NewClusterStore(suite.db)
	suite.tested, err = cs.CreateFlowStore("fakecluster")
	suite.Require().NoError(err)
}

func (suite *FlowStoreTestSuite) TearDownSuite() {
	_ = suite.db.Close()
	_ = os.RemoveAll(suite.path)
}

func (suite *FlowStoreTestSuite) TestStore() {
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someNode2"},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode1"},
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "someOtherNode2"},
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		},
	}
	var err error

	err = suite.tested.UpsertFlows(flows, timestamp.Now())
	suite.NoError(err, "upsert should succeed on first insert")

	err = suite.tested.UpsertFlows(flows, timestamp.Now())
	suite.NoError(err, "upsert should succeed on second insert")

	err = suite.tested.RemoveFlow(&storage.NetworkFlowProperties{
		SrcEntity:  flows[1].GetProps().GetSrcEntity(),
		DstEntity:  flows[1].GetProps().GetDstEntity(),
		DstPort:    flows[1].GetProps().GetDstPort(),
		L4Protocol: flows[1].GetProps().GetL4Protocol(),
	})
	suite.NoError(err, "remove should succeed when present")

	err = suite.tested.RemoveFlow(&storage.NetworkFlowProperties{
		SrcEntity:  flows[1].GetProps().GetSrcEntity(),
		DstEntity:  flows[1].GetProps().GetDstEntity(),
		DstPort:    flows[1].GetProps().GetDstPort(),
		L4Protocol: flows[1].GetProps().GetL4Protocol(),
	})
	suite.NoError(err, "remove should succeed when not present")

	var actualFlows []*storage.NetworkFlow
	actualFlows, _, err = suite.tested.GetAllFlows(nil)
	suite.Equal(1, len(actualFlows), "only flows[0] should be present")
	suite.Equal(flows[0], actualFlows[0], "flows should be equal")

	err = suite.tested.UpsertFlows(flows, timestamp.Now())
	suite.NoError(err, "upsert should succeed")

	actualFlows, _, err = suite.tested.GetAllFlows(nil)
	suite.Equal(2, len(actualFlows), "expected number of flows does not match")
	suite.ElementsMatch(flows, actualFlows, "upserted values should be as expected")

}
