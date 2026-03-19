package detection

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	mocks2 "github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/mocks"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type compiledPolicyWrapper struct {
	detection.CompiledPolicy

	policy *storage.Policy
}

func wrapPolicy(policy *storage.Policy) compiledPolicyWrapper {
	return compiledPolicyWrapper{
		policy: policy,
	}
}

func (w compiledPolicyWrapper) Policy() *storage.Policy {
	return w.policy
}

func TestPolicySet_RemoveNotifier(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	policySetMock := mocks.NewMockPolicySet(mockCtrl)
	policyDatastoreMock := mocks2.NewMockDataStore(mockCtrl)

	policySet := &setImpl{
		PolicySet:   policySetMock,
		policyStore: policyDatastoreMock,
	}

	policySetMock.EXPECT().GetCompiledPolicies().Return(map[string]detection.CompiledPolicy{
		"policy1": wrapPolicy(&storage.Policy{
			Id:        "policy1",
			Notifiers: []string{"notifier1", "notifier2"},
		}),
		"policy2": wrapPolicy(&storage.Policy{
			Id:        "policy2",
			Notifiers: []string{"notifier2", "notifier3"},
		}),
		"policy3": wrapPolicy(&storage.Policy{
			Id:        "policy3",
			Notifiers: []string{"notifier1", "notifier2", "notifier3"},
		}),
		"policy4": wrapPolicy(&storage.Policy{
			Id:        "policy4",
			Notifiers: []string{"notifier1", "notifier3"},
		}),
	})

	var updatedPolicies []*storage.Policy
	policyDatastoreMock.EXPECT().UpdatePolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, policy *storage.Policy) error {
		updatedPolicies = append(updatedPolicies, policy)
		return nil
	})

	require.NoError(t, policySet.RemoveNotifier("notifier2"))

	expectedUpdates := []*storage.Policy{
		{
			Id:        "policy1",
			Notifiers: []string{"notifier1"},
		},
		{
			Id:        "policy2",
			Notifiers: []string{"notifier3"},
		},
		{
			Id:        "policy3",
			Notifiers: []string{"notifier1", "notifier3"},
		},
	}

	protoassert.ElementsMatch(t, expectedUpdates, updatedPolicies)
}

func TestPolicySet_WithLabelProviders(t *testing.T) {
	testutils.MustUpdateFeature(t, features.LabelBasedPolicyScoping, true)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create mock datastores
	clusterDS := clusterMocks.NewMockDataStore(mockCtrl)
	namespaceDS := namespaceMocks.NewMockDataStore(mockCtrl)
	policyDS := mocks2.NewMockDataStore(mockCtrl)

	// Set up test data
	testClusterID := "test-cluster-123"
	testNamespaceID := "test-namespace-456"

	testCluster := &storage.Cluster{
		Id:   testClusterID,
		Name: "production-cluster",
		Labels: map[string]string{
			"env":    "prod",
			"region": "us-east-1",
		},
	}

	testNamespace := &storage.NamespaceMetadata{
		Id:          testNamespaceID,
		Name:        "backend-services",
		ClusterId:   testClusterID,
		ClusterName: "production-cluster",
		Labels: map[string]string{
			"team": "backend",
			"tier": "production",
		},
	}

	// Configure mock expectations
	clusterDS.EXPECT().
		GetCluster(gomock.Any(), testClusterID).
		Return(testCluster, true, nil).
		AnyTimes()

	namespaceDS.EXPECT().
		GetNamespace(gomock.Any(), testNamespaceID).
		Return(testNamespace, true, nil).
		AnyTimes()

	// Create PolicySet - datastores implement the provider interfaces directly
	policySet := NewPolicySet(policyDS, clusterDS, namespaceDS)

	// Test 1: Policy with cluster_label scope
	policyWithClusterLabel := &storage.Policy{
		Id:       "policy-cluster-label",
		Name:     "Production Cluster Policy",
		Severity: storage.Severity_HIGH_SEVERITY,
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageTag,
						Values: []*storage.PolicyValue{
							{Value: "latest"},
						},
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicyVersion:   "1.1",
	}

	err := policySet.UpsertPolicy(policyWithClusterLabel)
	require.NoError(t, err, "Policy with cluster_label should compile successfully")

	// Verify the policy was added
	assert.True(t, policySet.Exists("policy-cluster-label"))

	// Test 2: Policy with namespace_label scope
	policyWithNamespaceLabel := &storage.Policy{
		Id:       "policy-namespace-label",
		Name:     "Backend Team Policy",
		Severity: storage.Severity_MEDIUM_SEVERITY,
		Scope: []*storage.Scope{
			{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageTag,
						Values: []*storage.PolicyValue{
							{Value: ".*"},
						},
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicyVersion:   "1.1",
	}

	err = policySet.UpsertPolicy(policyWithNamespaceLabel)
	require.NoError(t, err, "Policy with namespace_label should compile successfully")

	assert.True(t, policySet.Exists("policy-namespace-label"))

	// Test 3: Policy with both cluster_label and namespace_label
	policyWithBothLabels := &storage.Policy{
		Id:       "policy-both-labels",
		Name:     "Prod Backend Policy",
		Severity: storage.Severity_CRITICAL_SEVERITY,
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.PrivilegedContainer,
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicyVersion:   "1.1",
	}

	err = policySet.UpsertPolicy(policyWithBothLabels)
	require.NoError(t, err, "Policy with both cluster_label and namespace_label should compile successfully")

	assert.True(t, policySet.Exists("policy-both-labels"))

	// Verify all three policies are in the set
	compiledPolicies := policySet.GetCompiledPolicies()
	assert.Len(t, compiledPolicies, 3, "PolicySet should contain all three policies")

	// Verify we can iterate over policies
	policyCount := 0
	err = policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		policyCount++
		assert.NotNil(t, compiled.Policy())
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, policyCount, "ForEach should iterate over all policies")
}
