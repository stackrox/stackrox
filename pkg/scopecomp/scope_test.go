package scopecomp

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithinScope(t *testing.T) {
	subtests := []struct {
		name            string
		scope           *storage.Scope
		deployment      *storage.Deployment
		clusterLabels   map[string]string
		namespaceLabels map[string]string
		result          bool
	}{
		{
			name:       "empty scope",
			scope:      &storage.Scope{},
			deployment: &storage.Deployment{},
			result:     true,
		},
		{
			name: "matching cluster",
			scope: &storage.Scope{
				Cluster: "cluster",
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			result: true,
		},
		{
			name: "not matching cluster",
			scope: &storage.Scope{
				Cluster: "cluster1",
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			result: false,
		},
		{
			name: "matching namespace",
			scope: &storage.Scope{
				Namespace: "namespace",
			},
			deployment: &storage.Deployment{
				Namespace: "namespace",
			},
			result: true,
		},
		{
			name: "not matching namespace",
			scope: &storage.Scope{
				Namespace: "namespace1",
			},
			deployment: &storage.Deployment{
				Namespace: "namespace",
			},
			result: false,
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
			result: true,
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
			result: true,
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
			result: false,
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
			result: false,
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
			result: true,
		},
		{
			name: "cluster label key-value matches",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels: map[string]string{
				"env": "prod",
			},
			result: true,
		},
		{
			name: "cluster label key-value does not match",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels: map[string]string{
				"env": "dev",
			},
			result: false,
		},
		{
			name: "namespace label key-value matches",
			scope: &storage.Scope{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "platform",
				},
			},
			deployment: &storage.Deployment{
				Namespace: "backend",
			},
			namespaceLabels: map[string]string{
				"team": "platform",
			},
			result: true,
		},
		{
			name: "namespace label key-value does not match",
			scope: &storage.Scope{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "platform",
				},
			},
			deployment: &storage.Deployment{
				Namespace: "backend",
			},
			namespaceLabels: map[string]string{
				"team": "frontend",
			},
			result: false,
		},
		{
			name: "cluster label with regex matches",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod|staging",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels: map[string]string{
				"env": "staging",
			},
			result: true,
		},
		{
			name: "cluster label filter set but labels nil - fail closed",
			scope: &storage.Scope{
				ClusterLabel: &storage.Scope_Label{
					Key: "env",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			clusterLabels: nil,
			result:        false,
		},
		{
			name: "namespace label filter set but labels nil - fail closed",
			scope: &storage.Scope{
				NamespaceLabel: &storage.Scope_Label{
					Key: "team",
				},
			},
			deployment: &storage.Deployment{
				Namespace: "backend",
			},
			namespaceLabels: nil,
			result:          false,
		},
		{
			name: "combined cluster and namespace labels match",
			scope: &storage.Scope{
				Cluster:   "cluster",
				Namespace: "backend",
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "platform",
				},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
				Namespace: "backend",
			},
			clusterLabels: map[string]string{
				"env": "prod",
			},
			namespaceLabels: map[string]string{
				"team": "platform",
			},
			result: true,
		},
	}
	for _, test := range subtests {
		cs, err := CompileScope(test.scope)
		require.NoError(t, err)
		assert.Equalf(t, test.result, cs.MatchesDeployment(test.deployment, test.clusterLabels, test.namespaceLabels), "Failed test '%s'", test.name)
	}
}
