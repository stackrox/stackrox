package secrets

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	ctx      context.Context
	clusters *clusterMocks.DataStore
	secrets  *secretMocks.DataStore
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.clusters = &clusterMocks.DataStore{}
	suite.secrets = &secretMocks.DataStore{}
}

func (suite *PipelineTestSuite) TestRun() {
	secret := fixtures.GetSecret()

	suite.clusters.On("GetCluster", "clusterid").Return(&v1.Cluster{Id: "clusterid", Name: "clustername"}, true, nil)
	suite.secrets.On("UpsertSecret", secret).Return(nil)

	pipeline := NewPipeline(suite.clusters, suite.secrets)
	sensorEvent := &v1.SensorEvent{
		Id:        "secretid",
		ClusterId: "clusterid",
		Action:    v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Secret{
			Secret: secret,
		},
	}
	err := pipeline.Run(sensorEvent, nil)
	suite.NoError(err)
}
