package analysis

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/k8srbac"
	"github.com/stretchr/testify/assert"
)

var defaultLabelMap = map[string]string{
	k8srbac.DefaultLabel.Key: k8srbac.DefaultLabel.Value,
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
