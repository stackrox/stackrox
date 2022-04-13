package analysis

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestFindsBindingsWithoutRoles(t *testing.T) {
	inputRoles := []*storage.K8SRole{
		{
			Id:     "role0",
			Labels: defaultLabelMap, // Default binding, should be ignored
		},
		{
			Id: "role1",
		},
	}
	inputBindings := []*storage.K8SRoleBinding{
		{
			RoleId: "role0",
		},
		{
			RoleId: "role0",
			Labels: defaultLabelMap, // Default binding, should be ignored
		},
		{
			RoleId: "role1",
		},
		{
			RoleId: "role2",
		},
	}
	expected := []*storage.K8SRoleBinding{
		inputBindings[3],
	}

	assert.Equal(t, expected, getBindingsWithoutRoles(inputRoles, inputBindings))
}
