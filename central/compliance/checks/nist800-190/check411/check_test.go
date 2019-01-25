package check411

import (
	"context"
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

	domain = framework.NewComplianceDomain(testCluster, testNodes, testDeployments)

	cvssPolicyEnabledAndEnforced = storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		Fields: &storage.PolicyFields{
			Cvss: &storage.NumericalPolicy{
				Value: 7,
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	buildPolicyEnforced = storage.Policy{
		Id:                 uuid.NewV4().String(),
		Name:               "Sample Build time",
		LifecycleStages:    []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
	}

	cvssPolicyDisabled = storage.Policy{
		Id:                 uuid.NewV4().String(),
		Name:               "Foo",
		Disabled:           true,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	imageIntegration = storage.ImageIntegration{
		Name:       "Clairify",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER},
	}
)

func TestNIST411_Success(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[cvssPolicyEnabledAndEnforced.GetName()] = &cvssPolicyEnabledAndEnforced
	policies[buildPolicyEnforced.GetName()] = &buildPolicyEnforced

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return([]*storage.ImageIntegration{&imageIntegration})

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 3)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[2].Status)
}

func TestNIST411_Fail(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[cvssPolicyDisabled.GetName()] = &cvssPolicyDisabled
	policies[buildPolicyEnforced.GetName()] = &buildPolicyEnforced

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return([]*storage.ImageIntegration{})

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 3)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[1].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[2].Status)

}
