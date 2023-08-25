package check411

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

	cvssPolicyEnabledAndEnforced = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Foo",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.CVSS,
						Values: []*storage.PolicyValue{
							{
								Value: "7",
							},
						},
					},
				},
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
		PolicyVersion:      "1.1",
	}

	buildPolicyEnforced = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Sample Build time",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.CVSS,
						Values: []*storage.PolicyValue{
							{
								Value: "7",
							},
						},
					},
				},
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
		PolicyVersion:      "1.1",
	}

	cvssPolicyDisabled = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Foo",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Disabled:        true,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.CVSS,
						Values: []*storage.PolicyValue{
							{
								Value: "7",
							},
						},
					},
				},
			},
		},
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
		PolicyVersion:      "1.1",
	}

	buildPolicyDisabled = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Sample Build time",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.CVSS,
						Values: []*storage.PolicyValue{
							{
								Value: "7",
							},
						},
					},
				},
			},
		},
		Disabled:           true,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
		PolicyVersion:      "1.1",
	}

	imageIntegration = storage.ImageIntegration{
		Name:       "Clairify",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER},
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

func TestNIST411_Success(t *testing.T) {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := getPolicies(t, cvssPolicyEnabledAndEnforced, buildPolicyEnforced)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return([]*storage.ImageIntegration{&imageIntegration})

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), "standard", domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 4)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[2].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[2].Status)
}

func TestNIST411_Fail(t *testing.T) {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := getPolicies(t, cvssPolicyDisabled, buildPolicyDisabled)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return([]*storage.ImageIntegration{})

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), "standard", domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 4)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[1].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[2].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[3].Status)
}
