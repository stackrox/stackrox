package scopecomp

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithinScope(t *testing.T) {
	subtests := []struct {
		name       string
		scope      *storage.Scope
		deployment *storage.Deployment
		result     bool
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
				ClusterScope: &storage.Scope_Cluster{Cluster: "cluster"},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			result: true,
		},
		{
			name: "not matching cluster",
			scope: &storage.Scope{
				ClusterScope: &storage.Scope_Cluster{Cluster: "cluster1"},
			},
			deployment: &storage.Deployment{
				ClusterId: "cluster",
			},
			result: false,
		},
		{
			name: "matching namespace",
			scope: &storage.Scope{
				NamespaceScope: &storage.Scope_Namespace{Namespace: "namespace"},
			},
			deployment: &storage.Deployment{
				Namespace: "namespace",
			},
			result: true,
		},
		{
			name: "not matching namespace",
			scope: &storage.Scope{
				NamespaceScope: &storage.Scope_Namespace{Namespace: "namespace1"},
			},
			deployment: &storage.Deployment{
				Namespace: "namespace",
			},
			result: false,
		},
		{
			name: "matching cluster with no namespace scope",
			scope: &storage.Scope{
				ClusterScope: &storage.Scope_Cluster{Cluster: "cluster"},
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
				ClusterScope:   &storage.Scope_Cluster{Cluster: "cluster"},
				NamespaceScope: &storage.Scope_Namespace{Namespace: "namespace"},
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
	}
	for _, test := range subtests {
		cs, err := CompileScope(test.scope)
		require.NoError(t, err)
		assert.Equalf(t, test.result, cs.MatchesDeployment(test.deployment), "Failed test '%s'", test.name)
	}
}
