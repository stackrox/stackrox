package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPermissionSetYAMLTransformation(t *testing.T) {
	data := []byte(`
name: test-name
description: test-description
resource_to_access:
  a: READ_ACCESS
  b: READ_WRITE_ACCESS
`)
	ps := PermissionSet{}

	err := yaml.Unmarshal(data, &ps)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ps.Name)
	assert.Equal(t, "test-description", ps.Description)
	accessValueForA, hasA := ps.ResourceToAccess["a"]
	accessValueForB, hasB := ps.ResourceToAccess["b"]
	assert.True(t, hasA)
	assert.True(t, hasB)
	assert.Len(t, ps.ResourceToAccess, 2)
	assert.Equal(t, storage.Access_READ_ACCESS, storage.Access(accessValueForA))
	assert.Equal(t, storage.Access_READ_WRITE_ACCESS, storage.Access(accessValueForB))
}
