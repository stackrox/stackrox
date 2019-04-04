package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestFindsRoleswithoutBindings(t *testing.T) {
	inputRoles := []*storage.K8SRole{
		{
			Id: "role0",
			Rules: []*storage.PolicyRule{
				{
					ApiGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "create"},
				},
			},
		},
		{
			Id: "role1",
			Rules: []*storage.PolicyRule{
				{
					ApiGroups: []string{""},
					Resources: []string{"pods", "deployments"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
		{
			Id: "role2",
			Rules: []*storage.PolicyRule{
				{
					ApiGroups: []string{""},
					Resources: []string{"deployments"},
					Verbs:     []string{"list"},
				},
			},
		},
		{
			Id: "role3",
			Rules: []*storage.PolicyRule{
				{
					ApiGroups: []string{""},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
	}
	inputBindings := []*storage.K8SRoleBinding{
		{
			RoleId: "role1",
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "robot",
					Namespace: "stackrox",
				},
			},
		},
		{
			RoleId: "role2",
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_USER,
					Name:      "robot",
					Namespace: "stackrox",
				},
			},
		},
		{
			RoleId: "role3",
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_GROUP,
					Name:      "robot",
					Namespace: "stackrox",
				},
			},
		},
	}
	inputSubject := &storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Namespace: "stackrox",
		Name:      "robot",
	}
	expected := []*storage.PolicyRule{
		{
			ApiGroups: []string{""},
			Resources: []string{"pods", "deployments"},
			Verbs:     []string{"get", "list"},
		},
	}

	evaluator := NewEvaluator(inputRoles, inputBindings)
	assert.Equal(t, expected, evaluator.ForSubject(inputSubject))
}
