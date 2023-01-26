package declarativeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPermissionSetYAMLTransformation(t *testing.T) {
	data := []byte(`
name: test-name
description: test-description
rules:
  includedClusters:
    - clusterA
    - clusterB
  includedNamespaces:
    - cluster: clusterC
      namespace: namespaceC
    - cluster: clusterD
      namespace: namespaceD
  clusterLabelSelectors:
    - requirements:
      - key: a
        operator: IN
        values: [a, b, c]
      - key: b
        operator: NOT_IN
        values: [b, d]
    - requirements:
      - key: c
        operator: IN
        values: [d, e, f]
      - key: d
        operator: NOT_IN
        values: [x, y, z]
`)
	as := AccessScope{}

	err := yaml.Unmarshal(data, &as)
	assert.NoError(t, err)
	// TODO: more asserts
	assert.Equal(t, "test-name", as.Name)
	assert.Equal(t, "test-description", as.Description)
}
