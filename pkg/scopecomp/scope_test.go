package scopecomp

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestWithinScope(t *testing.T) {
	subtests := []struct {
		name       string
		scope      *storage.Scope
		deployment *v1.Deployment
		result     bool
	}{
		{
			name:       "empty scope",
			scope:      &storage.Scope{},
			deployment: &v1.Deployment{},
			result:     true,
		},
		{
			name: "matching cluster",
			scope: &storage.Scope{
				Cluster: "cluster",
			},
			deployment: &v1.Deployment{
				ClusterId: "cluster",
			},
			result: true,
		},
		{
			name: "not matching cluster",
			scope: &storage.Scope{
				Cluster: "cluster1",
			},
			deployment: &v1.Deployment{
				ClusterId: "cluster",
			},
			result: false,
		},
		{
			name: "matching namespace",
			scope: &storage.Scope{
				Namespace: "namespace",
			},
			deployment: &v1.Deployment{
				Namespace: "namespace",
			},
			result: true,
		},
		{
			name: "not matching namespace",
			scope: &storage.Scope{
				Namespace: "namespace1",
			},
			deployment: &v1.Deployment{
				Namespace: "namespace",
			},
			result: false,
		},
		{
			name: "matching label",
			scope: &storage.Scope{
				Label: &storage.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &v1.Deployment{
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
			deployment: &v1.Deployment{
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
			deployment: &v1.Deployment{
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
			deployment: &v1.Deployment{
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
		assert.Equalf(t, test.result, WithinScope(test.scope, test.deployment), "Failed test '%s'", test.name)
	}
}
