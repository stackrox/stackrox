package declarativeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRoleYAMLTransformation(t *testing.T) {
	data := []byte(`name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
`)
	role := Role{}

	err := yaml.Unmarshal(data, &role)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", role.Name)
	assert.Equal(t, "test-description", role.Description)
	assert.Equal(t, "access-scope", role.AccessScope)
	assert.Equal(t, "permission-set", role.PermissionSet)

	bytes, err := yaml.Marshal(&role)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(bytes))
}
