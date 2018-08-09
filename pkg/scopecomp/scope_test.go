package scopecomp

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestWithinScope(t *testing.T) {
	subtests := []struct {
		name       string
		scope      *v1.Scope
		deployment *v1.Deployment
		result     bool
	}{
		{
			name:       "empty scope",
			scope:      &v1.Scope{},
			deployment: &v1.Deployment{},
			result:     true,
		},
		{
			name: "matching cluster",
			scope: &v1.Scope{
				Cluster: "cluster",
			},
			deployment: &v1.Deployment{
				ClusterId: "cluster",
			},
			result: true,
		},
		{
			name: "not matching cluster",
			scope: &v1.Scope{
				Cluster: "cluster1",
			},
			deployment: &v1.Deployment{
				ClusterId: "cluster",
			},
			result: false,
		},
		{
			name: "matching namespace",
			scope: &v1.Scope{
				Namespace: "namespace",
			},
			deployment: &v1.Deployment{
				Namespace: "namespace",
			},
			result: true,
		},
		{
			name: "not matching namespace",
			scope: &v1.Scope{
				Namespace: "namespace1",
			},
			deployment: &v1.Deployment{
				Namespace: "namespace",
			},
			result: false,
		},
		{
			name: "matching label",
			scope: &v1.Scope{
				Label: &v1.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
			},
			result: true,
		},
		{
			name: "not matching label value",
			scope: &v1.Scope{
				Label: &v1.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value1",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
			},
			result: false,
		},
		{
			name: "not matching key value",
			scope: &v1.Scope{
				Label: &v1.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key1",
						Value: "value",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
			},
			result: false,
		},
		{
			name: "match all",
			scope: &v1.Scope{
				Cluster:   "cluster",
				Namespace: "namespace",
				Label: &v1.Scope_Label{
					Key:   "key",
					Value: "value",
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "cluster",
				Namespace: "namespace",
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
			},
			result: true,
		},
	}
	for _, test := range subtests {
		assert.Equalf(t, test.result, WithinScope(test.scope, test.deployment), "Failed test '%s'", test.name)
	}
}
