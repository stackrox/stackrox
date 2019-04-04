package analysis

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var defaultLabelMap = map[string]string{
	defaultLabel.Key: defaultLabel.Value,
}

func TestFindsRoleswithoutBindings(t *testing.T) {
	inputRoles := []*storage.K8SRole{
		{
			Id:     "role0",
			Labels: defaultLabelMap, // Default binding, should be ignored
		},
		{
			Id: "role1",
		},
		{
			Id: "role2",
		},
		{
			Id: "role3",
		},
	}
	inputBindings := []*storage.K8SRoleBinding{
		{
			RoleId: "role1",
			Labels: defaultLabelMap, // Default binding, should be ignored
		},
		{
			RoleId: "role2",
		},
	}
	expected := []*storage.K8SRole{
		inputRoles[3],
	}

	assert.Equal(t, expected, getRolesWithoutBindings(inputRoles, inputBindings))
}
