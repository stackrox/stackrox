package check414

import (
	"context"
	"os"
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	complianceMocks "github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testCluster = &storage.Cluster{
		Id: uuid.NewV4().String(),
	}

	testNodes = []*storage.Node{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}
)

func TestNIST414_Success(t *testing.T) {
	f1, err := os.OpenFile("/tmp/username", os.O_CREATE, 0600)
	assert.NoError(t, err)

	f2, err := os.OpenFile("/tmp/passwd", os.O_CREATE, 0600)
	assert.NoError(t, err)

	defer func() {
		_ = os.Remove(f1.Name())
		_ = os.Remove(f2.Name())
	}()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	testDeployments := []*storage.Deployment{
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

	envSecretsEnabledAndEnforced := storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.EnvironmentVariable,
						Values: []*storage.PolicyValue{
							{
								Value: "=FOO_SECRET=34463",
							},
						},
					},
				},
			},
		},
		PolicyVersion:      "1.1",
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}
	envLowerSecretsEnabledAndEnforced := storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.EnvironmentVariable,
						Values: []*storage.PolicyValue{
							{
								Value: "=FOO_Secret_Blah=34463",
							},
						},
					},
				},
			},
		},
		PolicyVersion:      "1.1",
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	policies := make(map[string]*storage.Policy)
	policies[envSecretsEnabledAndEnforced.GetName()] = &envSecretsEnabledAndEnforced
	policies[envLowerSecretsEnabledAndEnforced.GetName()] = &envLowerSecretsEnabledAndEnforced

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().Deployments().AnyTimes().Return(toMapDeployments(testDeployments))

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil, nil)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), "standard", domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 2)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
}

func TestNIST414_FAIL(t *testing.T) {
	f1, err := os.OpenFile("/tmp/username", os.O_CREATE, 0644)
	assert.NoError(t, err)

	f2, err := os.OpenFile("/tmp/passwd", os.O_CREATE, 0644)
	assert.NoError(t, err)

	defer func() {
		_ = os.Remove(f1.Name())
		_ = os.Remove(f2.Name())
	}()

	testDeployments := []*storage.Deployment{
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

	envSecretsEnabledAndEnforced := storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.EnvironmentVariable,
						Values: []*storage.PolicyValue{
							{
								Value: "FOO_SECRET=34463",
							},
						},
					},
				},
			},
		},
		PolicyVersion:      "1.1",
		Disabled:           true,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}
	envLowerSecretsEnabledAndEnforced := storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.EnvironmentVariable,
						Values: []*storage.PolicyValue{
							{
								Value: "FOO_secret_Blah=34463",
							},
						},
					},
				},
			},
		},
		PolicyVersion:      "1.1",
		Disabled:           true,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

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
	data.EXPECT().Deployments().AnyTimes().Return(toMapDeployments(testDeployments))

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil, nil)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), "standard", domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 2)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[1].Status)

}

func toMapDeployments(in []*storage.Deployment) map[string]*storage.Deployment {
	merp := make(map[string]*storage.Deployment, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}
