package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestAccessScopeYAMLTransformation(t *testing.T) {
	data := []byte(`name: test-name
description: test-description
rules:
    included:
        - cluster: clusterA
          namespaces:
            - namespaceA1
            - namespaceA2
        - cluster: clusterB
    clusterLabelSelectors:
        - requirements:
            - key: a
              operator: IN
              values:
                - a
                - b
                - c
            - key: b
              operator: NOT_IN
              values:
                - b
                - d
        - requirements:
            - key: c
              operator: IN
              values:
                - d
                - e
                - f
            - key: d
              operator: NOT_IN
              values:
                - x
                - "y"
                - z
`)
	as := AccessScope{}

	err := yaml.Unmarshal(data, &as)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", as.Name)
	assert.Equal(t, "test-description", as.Description)

	assert.Len(t, as.Rules.IncludedObjects, 2)
	assert.Equal(t, as.Rules.IncludedObjects[0].Cluster, "clusterA")
	assert.Len(t, as.Rules.IncludedObjects[0].Namespaces, 2)
	assert.Equal(t, as.Rules.IncludedObjects[0].Namespaces[0], "namespaceA1")
	assert.Equal(t, as.Rules.IncludedObjects[0].Namespaces[1], "namespaceA2")
	assert.Equal(t, as.Rules.IncludedObjects[1].Cluster, "clusterB")
	assert.Len(t, as.Rules.IncludedObjects[1].Namespaces, 0)

	assert.Len(t, as.Rules.ClusterLabelSelectors, 2)
	assert.Len(t, as.Rules.ClusterLabelSelectors[0].Requirements, 2)
	operator := as.Rules.ClusterLabelSelectors[0].Requirements[0].Operator
	assert.Equal(t, storage.SetBasedLabelSelector_Operator(operator), storage.SetBasedLabelSelector_IN)
	assert.Equal(t, as.Rules.ClusterLabelSelectors[0].Requirements[0].Key, "a")
	assert.Equal(t, as.Rules.ClusterLabelSelectors[0].Requirements[0].Values, []string{"a", "b", "c"})
	operator = as.Rules.ClusterLabelSelectors[0].Requirements[1].Operator
	assert.Equal(t, storage.SetBasedLabelSelector_Operator(operator), storage.SetBasedLabelSelector_NOT_IN)
	assert.Equal(t, as.Rules.ClusterLabelSelectors[0].Requirements[1].Key, "b")
	assert.Equal(t, as.Rules.ClusterLabelSelectors[0].Requirements[1].Values, []string{"b", "d"})

	operator = as.Rules.ClusterLabelSelectors[1].Requirements[0].Operator
	assert.Equal(t, storage.SetBasedLabelSelector_Operator(operator), storage.SetBasedLabelSelector_IN)
	assert.Equal(t, as.Rules.ClusterLabelSelectors[1].Requirements[0].Key, "c")
	assert.Equal(t, as.Rules.ClusterLabelSelectors[1].Requirements[0].Values, []string{"d", "e", "f"})
	operator = as.Rules.ClusterLabelSelectors[1].Requirements[1].Operator
	assert.Equal(t, storage.SetBasedLabelSelector_Operator(operator), storage.SetBasedLabelSelector_NOT_IN)
	assert.Equal(t, as.Rules.ClusterLabelSelectors[1].Requirements[1].Key, "d")
	assert.Equal(t, as.Rules.ClusterLabelSelectors[1].Requirements[1].Values, []string{"x", "y", "z"})
	assert.Len(t, as.Rules.NamespaceLabelSelectors, 0)

	bytes, err := yaml.Marshal(&as)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(bytes))
}
