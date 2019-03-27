package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestLabelsMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		deployments    []*storage.Deployment
		expectedMap    map[string]*v1.DeploymentLabelsResponse_LabelValues
		expectedValues []string
	}{
		{
			name: "one deployment",
			deployments: []*storage.Deployment{
				{
					Labels: map[string]string{
						"key": "value",
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
			deployments: []*storage.Deployment{
				{
					Labels: map[string]string{
						"key":   "value",
						"hello": "world",
						"foo":   "bar",
					},
				},
				{
					Labels: map[string]string{
						"key": "hole",
						"app": "data",
						"foo": "bar",
					},
				},
				{
					Labels: map[string]string{
						"hello": "bob",
						"foo":   "boo",
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
			assert.ElementsMatch(t, c.expectedValues, actualValues)
		})
	}
}
