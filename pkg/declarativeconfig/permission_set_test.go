package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPermissionSetYAMLTransformation(t *testing.T) {
	data := []byte(`name: test-name
description: test-description
resources:
    - resource: a
      access: READ_ACCESS
    - resource: b
      access: READ_WRITE_ACCESS
`)
	ps := PermissionSet{}

	err := yaml.Unmarshal(data, &ps)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ps.Name)
	assert.Equal(t, "test-description", ps.Description)
	assert.Len(t, ps.Resources, 2)
	resourceA := ps.Resources[0]
	resourceB := ps.Resources[1]
	assert.Equal(t, "a", resourceA.Resource)
	assert.Equal(t, "b", resourceB.Resource)
	assert.Equal(t, storage.Access_READ_ACCESS, storage.Access(resourceA.Access))
	assert.Equal(t, storage.Access_READ_WRITE_ACCESS, storage.Access(resourceB.Access))

	bytes, err := yaml.Marshal(&ps)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(bytes))
}
