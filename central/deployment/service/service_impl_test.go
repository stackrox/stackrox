package service

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestLabelsMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		deployments    []*v1.Deployment
		expectedMap    map[string]*v1.DeploymentLabelsResponse_LabelValues
		expectedValues []string
	}{
		{
			name: "one deployment",
			deployments: []*v1.Deployment{
				{
					Labels: []*v1.Deployment_KeyValue{
						{
							Key:   "key",
							Value: "value",
						},
					},
				},
			},
			expectedMap: map[string]*v1.DeploymentLabelsResponse_LabelValues{
				"key": {
					Values: []string{"value"},
				},
			},
			expectedValues: []string{
				"value",
			},
		},
		{
			name: "multiple deployments",
			deployments: []*v1.Deployment{
				{
					Labels: []*v1.Deployment_KeyValue{
						{
							Key:   "key",
							Value: "value",
						},
						{
							Key:   "hello",
							Value: "world",
						},
						{
							Key:   "foo",
							Value: "bar",
						},
					},
				},
				{
					Labels: []*v1.Deployment_KeyValue{
						{
							Key:   "key",
							Value: "hole",
						},
						{
							Key:   "app",
							Value: "data",
						},
						{
							Key:   "foo",
							Value: "bar",
						},
					},
				},
				{
					Labels: []*v1.Deployment_KeyValue{
						{
							Key:   "hello",
							Value: "bob",
						},
						{
							Key:   "foo",
							Value: "boo",
						},
					},
				},
			},
			expectedMap: map[string]*v1.DeploymentLabelsResponse_LabelValues{
				"key": {
					Values: []string{"hole", "value"},
				},
				"hello": {
					Values: []string{"bob", "world"},
				},
				"foo": {
					Values: []string{"bar", "boo"},
				},
				"app": {
					Values: []string{"data"},
				},
			},
			expectedValues: []string{
				"bar", "bob", "boo", "data", "hole", "value", "world",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actualMap, actualValues := labelsMapFromDeployments(c.deployments)

			assert.Equal(t, c.expectedMap, actualMap)
			assert.Equal(t, c.expectedValues, actualValues)
		})
	}
}
