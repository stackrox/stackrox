package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/data"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	suite.Run(t, new(FlowStoreTestSuite))
}

type FlowStoreTestSuite struct {
	suite.Suite

	db     *bolt.DB
	tested FlowStore
}

func (suite *FlowStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.tested = NewFlowStore(db, "fakecluster")
}

func (suite *FlowStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *FlowStoreTestSuite) TestStore() {
	flows := []*data.NetworkFlow{
		{
			Props: &data.NetworkFlowProperties{
				SourceDeploymentId: "someNode1",
				TargetDeploymentId: "someNode2",
				TargetPort:         1,
			},
		},
		{
			Props: &data.NetworkFlowProperties{
				SourceDeploymentId: "someNode2",
				TargetDeploymentId: "someNode1",
				TargetPort:         2,
			},
		},
	}
	var err error

	err = suite.tested.AddFlow(flows[0])
	suite.NoError(err, "add should succeed for first insert")

	err = suite.tested.AddFlow(flows[0])
	suite.Error(err, "add should fail on second insert")

	err = suite.tested.UpdateFlow(flows[0])
	suite.NoError(err, "update should succeed on second insert")

	err = suite.tested.UpdateFlow(flows[1])
	suite.Error(err, "update should fail on first insert")

	err = suite.tested.UpsertFlow(flows[1])
	suite.NoError(err, "upsert should succeed on first insert")

	err = suite.tested.UpsertFlow(flows[1])
	suite.NoError(err, "upsert should succeed on second insert")

	err = suite.tested.RemoveFlow(&data.NetworkFlowProperties{
		SourceDeploymentId: flows[1].GetProps().GetSourceDeploymentId(),
		TargetDeploymentId: flows[1].GetProps().GetTargetDeploymentId(),
		TargetPort:         flows[1].GetProps().GetTargetPort(),
	})
	suite.NoError(err, "remove should succeed when present")

	err = suite.tested.RemoveFlow(&data.NetworkFlowProperties{
		SourceDeploymentId: flows[1].GetProps().GetSourceDeploymentId(),
		TargetDeploymentId: flows[1].GetProps().GetTargetDeploymentId(),
		TargetPort:         flows[1].GetProps().GetTargetPort(),
	})
	suite.NoError(err, "remove should succeed when not present")

	var actualFlows []*data.NetworkFlow
	actualFlows, err = suite.tested.GetAllFlows()
	suite.Equal(1, len(actualFlows), "only flows[0] should be present")
	suite.Equal(flows[0], actualFlows[0], "only flows[0] should be present")
}
