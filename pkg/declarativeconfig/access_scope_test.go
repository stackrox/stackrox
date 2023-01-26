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
    - namespace: namespaceD
  clusterLabelSelectors:
  	- requirements:
      - key: a
        operator: In
        values: [a, b, c]
      - key: b
        operator: NotIn
        values: [b, d]
    - requirements:
       - key: c
        operator: In
        values: [d, e, f]
      - key: d
        operator: NotIn
        values: [x, y, z]
`)
	as := AccessScope{}

	err := yaml.Unmarshal(data, &as)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", as.Name)
	assert.Equal(t, "test-description", as.Description)

}
