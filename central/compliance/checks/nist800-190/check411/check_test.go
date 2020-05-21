package check411

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	complianceMocks "github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
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

	cvssPolicyEnabledAndEnforced = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Foo",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Fields: &storage.PolicyFields{
			Cvss: &storage.NumericalPolicy{
				Value: 7,
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	buildPolicyEnforced = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Sample Build time",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Fields: &storage.PolicyFields{
			Cvss: &storage.NumericalPolicy{
				Value: 7,
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
	}

	cvssPolicyDisabled = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Foo",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Disabled:        true,
		Fields: &storage.PolicyFields{
			Cvss: &storage.NumericalPolicy{
				Value: 7,
			},
		},
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	buildPolicyDisabled = &storage.Policy{
		Id:              uuid.NewV4().String(),
		Name:            "Sample Build time",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Fields: &storage.PolicyFields{
			Cvss: &storage.NumericalPolicy{
				Value: 7,
			},
		},
		Disabled:           true,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
	}

	imageIntegration = storage.ImageIntegration{
		Name:       "Clairify",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER},
	}
)

func getPolicies(t *testing.T, policies ...*storage.Policy) map[string]*storage.Policy {
	m := make(map[string]*storage.Policy, len(policies))
	for _, p := range policies {
		if features.BooleanPolicyLogic.Enabled() {
			require.NoError(t, booleanpolicy.EnsureConverted(p))
		}
		m[p.GetName()] = p

	}
	return m
}

func TestNIST411_Success(t *testing.T) {
	testutils.RunWithAndWithoutFeatureFlag(t, features.BooleanPolicyLogic.EnvVar(), "", func(t *testing.T) {
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
		err = run.Run(context.Background(), domain, data)
		require.NoError(t, err)

		results := run.GetAllResults()
		checkResults := results[standardID]
		require.NotNil(t, checkResults)

		require.Len(t, checkResults.Evidence(), 4)
		assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
		assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
		assert.Equal(t, framework.PassStatus, checkResults.Evidence()[2].Status)
		assert.Equal(t, framework.PassStatus, checkResults.Evidence()[2].Status)
	})
}

func TestNIST411_Fail(t *testing.T) {
	testutils.RunWithAndWithoutFeatureFlag(t, features.BooleanPolicyLogic.EnvVar(), "", func(t *testing.T) {
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
	})
}
