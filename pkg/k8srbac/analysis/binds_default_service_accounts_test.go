package analysis

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stretchr/testify/assert"
)

func TestFindsBindingsForDefaultServiceAccounts(t *testing.T) {
	inputBindings := []*storage.K8SRoleBinding{
		{
			RoleId: "role",
			Labels: defaultLabelMap, // Default binding, should be ignored
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: k8srbac.DefaultServiceAccountName,
				},
			},
		},
		{
			RoleId: "role",
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: k8srbac.DefaultServiceAccountName,
				},
			},
		},
		{
			RoleId: "role",
		},
	}
	expected := []*storage.K8SRoleBinding{
		inputBindings[1],
	}

	assert.Equal(t, expected, getRoleBindingsForDefaultServiceAccounts(inputBindings))
}
