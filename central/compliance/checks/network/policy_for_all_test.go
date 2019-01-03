package network

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

func TestAllIngress_Success(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup("all-deployments-have-ingress-policy")
	require.NotNil(t, check)

	testCluster := &storage.Cluster{
		Id: uuid.NewV4().String(),
	}

	testDeployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	testNodes := []*storage.Node{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	data := mocks.NewMockComplianceDataRepository(mockCtrl)
	testPolicyID := uuid.NewV4().String()
	testPolicies := map[string]*storage.NetworkPolicy{
		testPolicyID: {
			Id: testPolicyID,
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					{},
				},
			},
		},
	}
	testNetworkGraph := &v1.NetworkGraph{
		Nodes: []*v1.NetworkNode{
			{
				Entity:    networkentity.ForDeployment(testDeployments[0].GetId()).ToProto(),
				PolicyIds: []string{testPolicies[testPolicyID].GetId()},
			},
			{
				Entity:    networkentity.ForDeployment(testDeployments[1].GetId()).ToProto(),
				PolicyIds: []string{testPolicies[testPolicyID].GetId()},
			},
		},
	}
	data.EXPECT().NetworkPolicies().AnyTimes().Return(testPolicies)
	data.EXPECT().NetworkGraph().AnyTimes().Return(testNetworkGraph)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results["all-deployments-have-ingress-policy"]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 0)
	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		require.NoError(t, deploymentResults.Error())
		require.Len(t, deploymentResults.Evidence(), 1)
		assert.Equal(t, framework.PassStatus, deploymentResults.Evidence()[0].Status)
	}
}
