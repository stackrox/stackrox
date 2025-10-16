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
			scope: storage.Scope_builder{
				Cluster: "cluster",
			}.Build(),
			deployment: storage.Deployment_builder{
				ClusterId: "cluster",
			}.Build(),
			result: true,
		},
		{
			name: "not matching cluster",
			scope: storage.Scope_builder{
				Cluster: "cluster1",
			}.Build(),
			deployment: storage.Deployment_builder{
				ClusterId: "cluster",
			}.Build(),
			result: false,
		},
		{
			name: "matching namespace",
			scope: storage.Scope_builder{
				Namespace: "namespace",
			}.Build(),
			deployment: storage.Deployment_builder{
				Namespace: "namespace",
			}.Build(),
			result: true,
		},
		{
			name: "not matching namespace",
			scope: storage.Scope_builder{
				Namespace: "namespace1",
			}.Build(),
			deployment: storage.Deployment_builder{
				Namespace: "namespace",
			}.Build(),
			result: false,
		},
		{
			name: "matching cluster with no namespace scope",
			scope: storage.Scope_builder{
				Cluster: "cluster",
			}.Build(),
			deployment: storage.Deployment_builder{
				ClusterId: "cluster",
				Namespace: "namespace",
			}.Build(),
			result: true,
		},
		{
			name: "matching label",
			scope: storage.Scope_builder{
				Label: storage.Scope_Label_builder{
					Key:   "key",
					Value: "value",
				}.Build(),
			}.Build(),
			deployment: storage.Deployment_builder{
				Labels: map[string]string{
					"key":  "value",
					"key2": "value2",
				},
			}.Build(),
			result: true,
		},
		{
			name: "not matching label value",
			scope: storage.Scope_builder{
				Label: storage.Scope_Label_builder{
					Key:   "key",
					Value: "value",
				}.Build(),
			}.Build(),
			deployment: storage.Deployment_builder{
				Labels: map[string]string{
					"key":  "value1",
					"key2": "value2",
				},
			}.Build(),
			result: false,
		},
		{
			name: "not matching key value",
			scope: storage.Scope_builder{
				Label: storage.Scope_Label_builder{
					Key:   "key",
					Value: "value",
				}.Build(),
			}.Build(),
			deployment: storage.Deployment_builder{
				Labels: map[string]string{
					"key":  "value1",
					"key2": "value2",
				},
			}.Build(),
			result: false,
		},
		{
			name: "match all",
			scope: storage.Scope_builder{
				Cluster:   "cluster",
				Namespace: "namespace",
				Label: storage.Scope_Label_builder{
					Key:   "key",
					Value: "value",
				}.Build(),
			}.Build(),
			deployment: storage.Deployment_builder{
				ClusterId: "cluster",
				Namespace: "namespace",
				Labels: map[string]string{
					"key":  "value",
					"key2": "value2",
				},
			}.Build(),
			result: true,
		},
	}
	for _, test := range subtests {
		cs, err := CompileScope(test.scope)
		require.NoError(t, err)
		assert.Equalf(t, test.result, cs.MatchesDeployment(test.deployment), "Failed test '%s'", test.name)
	}
}
