package check422

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	complianceMocks "github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testCluster = &storage.Cluster{
		Id: uuid.NewV4().String(),
	}

	testDeployments = []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
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

	domain = framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil, nil)

	latestTagEnabledAndEnforced = &storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageTag,
						Values: []*storage.PolicyValue{
							{
								Value: "latest",
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

	imageAgePolicyEnabledAndEnforced = &storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Bar",
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageAge,
						Values: []*storage.PolicyValue{
							{
								Value: "30",
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

	randomPolicy = &storage.Policy{
		Id:       uuid.NewV4().String(),
		Name:     "Random",
		Disabled: false,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ProcessName,
						Values: []*storage.PolicyValue{
							{
								Value: "sshd",
							},
						},
					},
				},
			},
		},
		PolicyVersion:      "1.1",
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}
)

func getPolicies(t *testing.T, policies ...*storage.Policy) map[string]*storage.Policy {
	m := make(map[string]*storage.Policy, len(policies))
	for _, p := range policies {
		require.NoError(t, policyversion.EnsureConvertedToLatest(p))
		m[p.GetName()] = p
	}
	return m
}

func TestNIST422_Success(t *testing.T) {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := getPolicies(t, latestTagEnabledAndEnforced, imageAgePolicyEnabledAndEnforced)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)

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

func TestNIST422_Fail(t *testing.T) {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := getPolicies(t, randomPolicy)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)

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
