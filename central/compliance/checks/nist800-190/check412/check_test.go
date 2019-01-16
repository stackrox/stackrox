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
		Id:                 uuid.NewV4().String(),
		Name:               "CVSS >= 6 and Privileged",
		Categories:         []string{"Vulnerability Management", "Privileges"},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	sshPolicy = storage.Policy{
		Id:                 uuid.NewV4().String(),
		Name:               "Secure Shell (ssh) Port Exposed",
		Categories:         []string{"Security Best Practices"},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
	}

	privPolicyDisabled = storage.Policy{
		Id:                 uuid.NewV4().String(),
		Name:               "Privileged Container",
		Disabled:           true,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	imageIntegration = storage.ImageIntegration{
		Name: "Clairify",
	}
)

func TestNIST412_Success(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup("NIST-800-190:4.1.2")
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[cvssPolicyEnabledAndEnforced.GetName()] = &cvssPolicyEnabledAndEnforced
	policies[sshPolicy.GetName()] = &sshPolicy

	categoryPolicies := make(map[string]set.StringSet)
	policySet := set.NewStringSet()
	policySet.Add(cvssPolicyEnabledAndEnforced.Name)
	categoryPolicies["Vulnerability Management"] = policySet

	privSet := set.NewStringSet()
	privSet.Add(cvssPolicyEnabledAndEnforced.Name)
	categoryPolicies["Vulnerability Management"] = privSet

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().PolicyCategories().AnyTimes().Return(categoryPolicies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return([]*storage.ImageIntegration{&imageIntegration})

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results["NIST-800-190:4.1.2"]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 2)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
}

func TestNIST412_Fail(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup("NIST-800-190:4.1.2")
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[privPolicyDisabled.GetName()] = &privPolicyDisabled

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	categoryPolicies := make(map[string]set.StringSet)
	privSet := set.NewStringSet()
	privSet.Add(privPolicyDisabled.Name)
	categoryPolicies["Privileges"] = privSet

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)
	data.EXPECT().PolicyCategories().AnyTimes().Return(categoryPolicies)
	data.EXPECT().ImageIntegrations().AnyTimes().Return([]*storage.ImageIntegration{})

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results["NIST-800-190:4.1.2"]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 3)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[2].Status)
}
