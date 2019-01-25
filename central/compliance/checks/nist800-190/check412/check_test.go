package check412

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	complianceMocks "github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCluster = &storage.Cluster{
		Id: uuid.NewV4().String(),
	}

	testDeployments = []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				{
					Name: "container1",
				},
			},
		},
		{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				{
					Name: "container2",
				},
			},
		},
	}

	testNodes = []*storage.Node{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	domain = framework.NewComplianceDomain(testCluster, testNodes, testDeployments)

	cvssPolicyEnabledAndEnforced = storage.Policy{
		Id:                 uuid.NewV4().String(),
		Name:               "CVSS >= 6 and Privileged",
		Categories:         []string{"Vulnerability Management", "Privileges"},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	buildPolicy = storage.Policy{
		Id:                 uuid.NewV4().String(),
		Name:               "Sample build time policy",
		LifecycleStages:    []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	indicatorsWithSSH = []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  testDeployments[0].GetId(),
			ContainerName: testDeployments[0].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:          15,
				Name:         "ssh",
				ExecFilePath: "/usr/bin/ssh",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  testDeployments[1].GetId(),
			ContainerName: testDeployments[1].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:          32,
				Name:         "sshd",
				ExecFilePath: "/bin/sshd",
			},
		},
	}

	indicatorsWithoutSSH = []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  testDeployments[0].GetId(),
			ContainerName: testDeployments[0].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:          15,
				Name:         "ssh",
				ExecFilePath: "/bin/bash",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  testDeployments[1].GetId(),
			ContainerName: testDeployments[1].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:          32,
				Name:         "sshd",
				ExecFilePath: "/bin/zsh",
			},
		},
	}

	privPolicyDisabled = storage.Policy{
		Id:                 uuid.NewV4().String(),
		Name:               "Privileged Container",
		Disabled:           true,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	imageIntegrations = []*storage.ImageIntegration{
		{
			Name: "Clairify",
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_SCANNER,
			},
		},
		{
			Name: "DTR",
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_REGISTRY,
			},
		},
	}

	sshPolicy = storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "SSH Policy",
		Fields: &storage.PolicyFields{
			ProcessPolicy: &storage.ProcessPolicy{
				Name: "sshd",
			},
		},
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	sshPolicyWithNoEnforcement = storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "SSH Policy",
		Fields: &storage.PolicyFields{
			ProcessPolicy: &storage.ProcessPolicy{
				Name: "sshd",
			},
		},
	}
)

func TestNIST412_Success(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[cvssPolicyEnabledAndEnforced.GetName()] = &cvssPolicyEnabledAndEnforced
	policies[sshPolicy.GetName()] = &sshPolicy
	policies[buildPolicy.GetName()] = &buildPolicy

	categoryPolicies := make(map[string]set.StringSet)
	policySet := set.NewStringSet()
	policySet.Add(cvssPolicyEnabledAndEnforced.Name)
	categoryPolicies["Vulnerability Management"] = policySet

	privSet := set.NewStringSet()
	privSet.Add(cvssPolicyEnabledAndEnforced.Name)
	categoryPolicies["Privileges"] = privSet

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().PolicyCategories().AnyTimes().Return(categoryPolicies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return(imageIntegrations)
	data.EXPECT().ProcessIndicators().AnyTimes().Return(indicatorsWithoutSSH)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)
	require.Len(t, checkResults.Evidence(), 4)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[2].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[3].Status)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		assert.NoError(t, deploymentResults.Error())
		assert.Len(t, deploymentResults.Evidence(), 1)
		assert.Equal(t, framework.PassStatus, deploymentResults.Evidence()[0].Status)
	}
}

func TestNIST412_FAIL(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[privPolicyDisabled.GetName()] = &privPolicyDisabled
	policies[sshPolicy.GetName()] = &sshPolicyWithNoEnforcement

	categoryPolicies := make(map[string]set.StringSet)
	policySet := set.NewStringSet()
	policySet.Add(cvssPolicyEnabledAndEnforced.Name)
	categoryPolicies["Vulnerability Management"] = policySet

	privSet := set.NewStringSet()
	privSet.Add(cvssPolicyEnabledAndEnforced.Name)
	categoryPolicies["Privileges"] = privSet

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().PolicyCategories().AnyTimes().Return(categoryPolicies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return(nil)
	data.EXPECT().ProcessIndicators().AnyTimes().Return(indicatorsWithSSH)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)
	require.Len(t, checkResults.Evidence(), 4)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[1].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[2].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[3].Status)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		assert.NoError(t, deploymentResults.Error())
		assert.Len(t, deploymentResults.Evidence(), 1)
		assert.Equal(t, framework.FailStatus, deploymentResults.Evidence()[0].Status)
	}
}

func toMap(in []*storage.Deployment) map[string]*storage.Deployment {
	merp := make(map[string]*storage.Deployment, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}
