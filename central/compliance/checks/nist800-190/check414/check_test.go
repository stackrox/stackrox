package check414

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
)

func TestNIST414_Success(t *testing.T) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := make(map[string]*storage.Policy)
	policies[envSecretsEnabledAndEnforced.GetName()] = &envSecretsEnabledAndEnforced

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
}
