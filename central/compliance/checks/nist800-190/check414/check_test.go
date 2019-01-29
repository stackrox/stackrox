package check414

import (
	"context"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	complianceMocks "github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
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
			Id:   uuid.NewV4().String(),
			Name: "container1",
			Containers: []*storage.Container{
				{
					Volumes: []*storage.Volume{
						{
							Name:        "username",
							Destination: "/tmp/",
							Type:        "secret",
						},
					},
				},
			},
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "container2",
			Containers: []*storage.Container{
				{
					Volumes: []*storage.Volume{
						{
							Name:        "passwd",
							Destination: "/tmp/",
							Type:        "secret",
						},
					},
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

	envSecretsEnabledAndEnforced = storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		Fields: &storage.PolicyFields{
			Env: &storage.KeyValuePolicy{
				Key:   "FOO_SECRET",
				Value: "34463",
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}
	envLowerSecretsEnabledAndEnforced = storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		Fields: &storage.PolicyFields{
			Env: &storage.KeyValuePolicy{
				Key:   "FOO_secret_Blah",
				Value: "34463",
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}
)

func TestNIST414_Success(t *testing.T) {
	f1, err := os.OpenFile("/tmp/username", os.O_CREATE, 0600)
	assert.NoError(t, err)

	f2, err := os.OpenFile("/tmp/passwd", os.O_CREATE, 0600)
	assert.NoError(t, err)

	defer func() {
		os.Remove(f1.Name())
		os.Remove(f2.Name())
	}()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[envSecretsEnabledAndEnforced.GetName()] = &envSecretsEnabledAndEnforced
	policies[envLowerSecretsEnabledAndEnforced.GetName()] = &envLowerSecretsEnabledAndEnforced

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 1)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		assert.NoError(t, deploymentResults.Error())
		require.Len(t, deploymentResults.Evidence(), 1)
		assert.Equal(t, framework.PassStatus, deploymentResults.Evidence()[0].Status)
	}
}

func TestNIST414_FAIL(t *testing.T) {
	f1, err := os.OpenFile("/tmp/username", os.O_CREATE, 0644)
	assert.NoError(t, err)

	f2, err := os.OpenFile("/tmp/passwd", os.O_CREATE, 0644)
	assert.NoError(t, err)

	defer func() {
		os.Remove(f1.Name())
		os.Remove(f2.Name())
	}()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[envSecretsEnabledAndEnforced.GetName()] = &envSecretsEnabledAndEnforced
	policies[envLowerSecretsEnabledAndEnforced.GetName()] = &envLowerSecretsEnabledAndEnforced

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 1)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		assert.NoError(t, deploymentResults.Error())
		require.Len(t, deploymentResults.Evidence(), 1)
		assert.Equal(t, framework.FailStatus, deploymentResults.Evidence()[0].Status)
	}
}
