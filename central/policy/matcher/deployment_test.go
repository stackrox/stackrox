package matcher

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDeploymentMatcher(t *testing.T) {
	cases := []struct {
		policy     *storage.Policy
		deployment *storage.Deployment
		matches    bool
	}{
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster: "cluster1",
					},
				},
			},
			matches: true,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster:   "cluster2",
						Namespace: "ns1",
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Namespace: "ns1",
					},
				},
			},
			matches: true,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy:  &storage.Policy{},
			matches: true,
		},
	}

	for _, c := range cases {
		actual := NewDeploymentMatcher(c.deployment, nil, nil).IsPolicyApplicable(context.Background(), c.policy)
		assert.Equal(t, c.matches, actual)
	}
}

func TestDeploymentWithExclusion(t *testing.T) {
	cases := []struct {
		policy     *storage.Policy
		deployment *storage.Deployment
		matches    bool
	}{
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster: "cluster1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Scope: &storage.Scope{
								Namespace: "ns.*",
							},
						},
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster:   "cluster1",
						Namespace: "ns1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns.*",
							},
						},
					},
				},
			},
			matches: true,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Namespace: "ns1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment1",
						},
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Namespace: "ns1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
					{
						Deployment: &storage.Exclusion_Deployment{
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy:  &storage.Policy{},
			matches: true,
		},
	}

	for _, c := range cases {
		actual := NewDeploymentMatcher(c.deployment, nil, nil).IsPolicyApplicable(context.Background(), c.policy)
		assert.Equal(t, c.matches, actual)
	}
}

func TestDeploymentMatcher_WithLabelProviders(t *testing.T) {
	testutils.MustUpdateFeature(t, features.LabelBasedPolicyScoping, true)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	clusterDS := clusterMocks.NewMockDataStore(mockCtrl)
	namespaceDS := namespaceMocks.NewMockDataStore(mockCtrl)

	testCluster := &storage.Cluster{
		Id:   "cluster1",
		Name: "prod-cluster",
		Labels: map[string]string{
			"env":    "prod",
			"region": "us-east-1",
		},
	}

	testNamespace := &storage.NamespaceMetadata{
		Id:          "ns1",
		Name:        "kube-system",
		ClusterId:   "cluster1",
		ClusterName: "prod-cluster",
		Labels: map[string]string{
			"tier": "system",
		},
	}

	// Configure mock expectations
	clusterDS.EXPECT().GetCluster(gomock.Any(), "cluster1").Return(testCluster, true, nil).AnyTimes()
	clusterDS.EXPECT().GetClusterLabels(gomock.Any(), "cluster1").Return(testCluster.GetLabels(), nil).AnyTimes()
	namespaceDS.EXPECT().GetNamespace(gomock.Any(), "ns1").Return(testNamespace, true, nil).AnyTimes()
	namespaceDS.EXPECT().GetNamespaceLabels(gomock.Any(), "ns1").Return(testNamespace.GetLabels(), nil).AnyTimes()

	deployment := &storage.Deployment{
		Name:        "test-deployment",
		ClusterId:   "cluster1",
		Namespace:   "kube-system",
		NamespaceId: "ns1",
	}

	cases := []struct {
		name    string
		policy  *storage.Policy
		matches bool
	}{
		{
			name: "policy with matching cluster label",
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						ClusterLabel: &storage.Scope_Label{
							Key:   "env",
							Value: "prod",
						},
					},
				},
			},
			matches: true,
		},
		{
			name: "policy with non-matching cluster label",
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						ClusterLabel: &storage.Scope_Label{
							Key:   "env",
							Value: "dev",
						},
					},
				},
			},
			matches: false,
		},
		{
			name: "policy with matching namespace label",
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						NamespaceLabel: &storage.Scope_Label{
							Key:   "tier",
							Value: "system",
						},
					},
				},
			},
			matches: true,
		},
		{
			name: "policy with non-matching namespace label",
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						NamespaceLabel: &storage.Scope_Label{
							Key:   "tier",
							Value: "app",
						},
					},
				},
			},
			matches: false,
		},
		{
			name: "policy with both matching cluster and namespace labels",
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						ClusterLabel: &storage.Scope_Label{
							Key:   "region",
							Value: "us-east-1",
						},
						NamespaceLabel: &storage.Scope_Label{
							Key:   "tier",
							Value: "system",
						},
					},
				},
			},
			matches: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matcher := NewDeploymentMatcher(deployment, clusterDS, namespaceDS)
			actual := matcher.IsPolicyApplicable(context.Background(), c.policy)
			assert.Equal(t, c.matches, actual)
		})
	}
}
