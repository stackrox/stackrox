package scopecomp

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scopecomp/mocks"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWithinScope(t *testing.T) {
	subtests := []struct {
		name               string
		scope              *storage.Scope
		deployment         *storage.Deployment
		clusterLabels      map[string]string
		namespaceLabels    map[string]string
		featureFlagEnabled bool
		result             bool
	}{
		{
			name:               "empty scope",
			scope:              &storage.Scope{},
			deployment:         &storage.Deployment{},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "matching cluster",
			scope: &storage.Scope{
				Cluster: "cluster",
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "not matching cluster",
			scope: &storage.Scope{
				Cluster: "cluster1",
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			featureFlagEnabled: false,
			result:             false,
		},
		{
			name: "matching namespace",
			scope: &storage.Scope{
				Namespace: "namespace",
			},
			deployment: &storage.Deployment{
				Namespace: "namespace",
			},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "not matching namespace",
			scope: &storage.Scope{
				Namespace: "namespace1",
			},
			deployment: &storage.Deployment{
				Namespace: "namespace",
			},
			featureFlagEnabled: false,
			result:             false,
		},
		{
			name: "matching cluster with no namespace scope",
			scope: &storage.Scope{
				Cluster: "cluster",
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
				Namespace: "namespace",
			},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "matching label",
			scope: &storage.Scope{
				Label: &storage.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &storage.Deployment{
				Labels: map[string]string{
					"key":  "value",
					"key2": "value2",
				},
			},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "not matching label value",
			scope: &storage.Scope{
				Label: &storage.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &storage.Deployment{
				Labels: map[string]string{
					"key":  "value1",
					"key2": "value2",
				},
			},
			featureFlagEnabled: false,
			result:             false,
		},
		{
			name: "not matching key value",
			scope: &storage.Scope{
				Label: &storage.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &storage.Deployment{
				Labels: map[string]string{
					"key":  "value1",
					"key2": "value2",
				},
			},
			featureFlagEnabled: false,
			result:             false,
		},
		{
			name: "match all",
			scope: &storage.Scope{
				Cluster:   "cluster",
				Namespace: "namespace",
				Label: &storage.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
				Namespace: "namespace",
				Labels: map[string]string{
					"key":  "value",
					"key2": "value2",
				},
			},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "scope with cluster_label",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels:      map[string]string{"env": "prod"},
			featureFlagEnabled: true,
			result:             true,
		},
		{
			name: "scope with namespace_label",
			scope: &storage.Scope{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
			deployment: &storage.Deployment{
				Namespace: "default",
			},
			namespaceLabels:    map[string]string{"team": "backend"},
			featureFlagEnabled: true,
			result:             true,
		},
		{
			name: "scope with cluster_label and namespace_label",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
				Namespace: "default",
			},
			clusterLabels:      map[string]string{"env": "prod"},
			namespaceLabels:    map[string]string{"team": "backend"},
			featureFlagEnabled: true,
			result:             true,
		},
		// Test cases verifying feature flag behavior
		{
			name: "cluster_label mismatch with flag OFF is ignored",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels:      map[string]string{"env": "dev"},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "cluster_label mismatch with flag ON fails",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels:      map[string]string{"env": "dev"},
			featureFlagEnabled: true,
			result:             false,
		},
		{
			name: "namespace_label mismatch with flag OFF is ignored",
			scope: &storage.Scope{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
			deployment: &storage.Deployment{
				Namespace: "default",
			},
			namespaceLabels:    map[string]string{"team": "frontend"},
			featureFlagEnabled: false,
			result:             true,
		},
		{
			name: "namespace_label mismatch with flag ON fails",
			scope: &storage.Scope{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
			deployment: &storage.Deployment{
				Namespace: "default",
			},
			namespaceLabels:    map[string]string{"team": "frontend"},
			featureFlagEnabled: true,
			result:             false,
		},
		// Test cases for nil provider handling
		{
			name: "nil providers with no label matchers should pass",
			scope: &storage.Scope{
				Cluster:   "cluster",
				Namespace: "namespace",
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
				Namespace: "namespace",
			},
			clusterLabels:      nil,
			namespaceLabels:    nil,
			featureFlagEnabled: true,
			result:             true,
		},
		{
			name: "nil cluster provider with cluster_label matcher should fail",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels:      nil,
			namespaceLabels:    nil,
			featureFlagEnabled: true,
			result:             false,
		},
		{
			name: "nil namespace provider with namespace_label matcher should fail",
			scope: &storage.Scope{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
			deployment: &storage.Deployment{
				Namespace: "default",
			},
			clusterLabels:      nil,
			namespaceLabels:    nil,
			featureFlagEnabled: true,
			result:             false,
		},
	}
	for _, test := range subtests {
		testutils.MustUpdateFeature(t, features.LabelBasedPolicyScoping, test.featureFlagEnabled)

		// Create gomock controller and mock providers
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		var clusterProvider ClusterLabelProvider
		var namespaceProvider NamespaceLabelProvider
		if test.clusterLabels != nil {
			mockCluster := mocks.NewMockClusterLabelProvider(ctrl)
			mockCluster.EXPECT().
				GetClusterLabels(gomock.Any(), gomock.Any()).
				Return(test.clusterLabels, nil).
				AnyTimes()
			clusterProvider = mockCluster
		}
		if test.namespaceLabels != nil {
			mockNamespace := mocks.NewMockNamespaceLabelProvider(ctrl)
			mockNamespace.EXPECT().
				GetNamespaceLabels(gomock.Any(), gomock.Any()).
				Return(test.namespaceLabels, nil).
				AnyTimes()
			namespaceProvider = mockNamespace
		}

		cs, err := CompileScope(test.scope, clusterProvider, namespaceProvider)
		require.NoError(t, err)
		assert.Equalf(t, test.result, cs.MatchesDeployment(context.Background(), test.deployment), "Failed test '%s'", test.name)
	}
}
