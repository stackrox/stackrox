package check112

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkentity"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck112(t *testing.T) {
	t.Parallel()
	registry := framework.RegistrySingleton()
	checkName := "PCI_DSS_3_2:1_1_2"
	check := registry.Lookup(checkName)
	require.NotNil(t, check)

	testCluster := &storage.Cluster{
		Id: uuid.NewV4().String(),
	}

	testDeployments := []*storage.Deployment{
		{Id: uuid.NewV4().String()},
		{Id: uuid.NewV4().String()},
	}

	testNodes := []*storage.Node{
		{Id: uuid.NewV4().String()},
		{Id: uuid.NewV4().String()},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	data := mocks.NewMockComplianceDataRepository(mockCtrl)

	testNetworkGraph := &v1.NetworkGraph{
		Nodes: []*v1.NetworkNode{
			{Entity: networkentity.ForDeployment(testDeployments[0].GetId()).ToProto()},
			{Entity: networkentity.ForDeployment(testDeployments[1].GetId()).ToProto()},
		},
	}
	data.EXPECT().NetworkGraph().AnyTimes().Return(testNetworkGraph)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[checkName]
	require.NotNil(t, checkResults)
	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		require.NoError(t, deploymentResults.Error())
		require.Len(t, deploymentResults.Evidence(), 1)
		assert.Equal(t, framework.PassStatus, deploymentResults.Evidence()[0].Status)
	}
}
