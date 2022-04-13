package k8srbac

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestFindsBindingsForClusterAdmin(t *testing.T) {
	inputRoles := []*storage.K8SRole{
		{
			Id:          "role1",
			Name:        clusterAdmin,
			ClusterRole: true,
		},
		{
			Id:          "role2",
			Name:        "some other name",
			ClusterRole: true,
		},
		{
			Id:          "role3",
			Name:        "effective admin",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				{
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				},
			},
		},
		{
			Id:          "role4",
			Name:        "another effective admin",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				{
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				},
			},
		},
		{
			Id:          "role5",
			Name:        "can get anything",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				{
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"Get",
					},
				},
			},
		},
	}
	inputBindings := []*storage.K8SRoleBinding{
		{
			RoleId: "role1",
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			}, // Default binding, should be ignored
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: DefaultServiceAccountName,
				},
			},
		},
		{
			RoleId: "role1",
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "admin",
				},
			},
		},
		{
			RoleId: "role2",
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "some non admin account",
				},
			},
		},
		{
			RoleId: "role3",
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "effective admin",
				},
			},
		},
		{
			RoleId: "role4",
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "another effective admin",
				},
			},
		},
		{
			RoleId: "role5",
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "analyst",
				},
			},
		},
	}
	expected := []*storage.Subject{
		{
			Kind: storage.SubjectKind_SERVICE_ACCOUNT,
			Name: "admin",
		},
		{
			Kind: storage.SubjectKind_SERVICE_ACCOUNT,
			Name: "another effective admin",
		},
		{
			Kind: storage.SubjectKind_SERVICE_ACCOUNT,
			Name: "effective admin",
		},
	}

	assert.Equal(t, expected, getSubjectsGrantedClusterAdmin(inputRoles, inputBindings))
}

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
	assert.Equal(t, expected, evaluator.ForSubject(inputSubject).ToSlice())
}
