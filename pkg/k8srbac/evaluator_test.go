package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

func TestFindsBindingsForClusterAdmin(t *testing.T) {
	inputRoles := []*storage.K8SRole{
		storage.K8SRole_builder{
			Id:          "role1",
			Name:        clusterAdmin,
			ClusterRole: true,
		}.Build(),
		storage.K8SRole_builder{
			Id:          "role2",
			Name:        "some other name",
			ClusterRole: true,
		}.Build(),
		storage.K8SRole_builder{
			Id:          "role3",
			Name:        "effective admin",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				storage.PolicyRule_builder{
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				}.Build(),
			},
		}.Build(),
		storage.K8SRole_builder{
			Id:          "role4",
			Name:        "another effective admin",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				storage.PolicyRule_builder{
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				}.Build(),
			},
		}.Build(),
		storage.K8SRole_builder{
			Id:          "role5",
			Name:        "can get anything",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				storage.PolicyRule_builder{
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"Get",
					},
				}.Build(),
			},
		}.Build(),
	}
	inputBindings := []*storage.K8SRoleBinding{
		storage.K8SRoleBinding_builder{
			RoleId: "role1",
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			}, // Default binding, should be ignored
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: DefaultServiceAccountName,
				}.Build(),
			},
		}.Build(),
		storage.K8SRoleBinding_builder{
			RoleId: "role1",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "admin",
				}.Build(),
			},
		}.Build(),
		storage.K8SRoleBinding_builder{
			RoleId: "role2",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "some non admin account",
				}.Build(),
			},
		}.Build(),
		storage.K8SRoleBinding_builder{
			RoleId: "role3",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "effective admin",
				}.Build(),
			},
		}.Build(),
		storage.K8SRoleBinding_builder{
			RoleId: "role4",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "another effective admin",
				}.Build(),
			},
		}.Build(),
		storage.K8SRoleBinding_builder{
			RoleId: "role5",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "analyst",
				}.Build(),
			},
		}.Build(),
	}
	subject := &storage.Subject{}
	subject.SetKind(storage.SubjectKind_SERVICE_ACCOUNT)
	subject.SetName("admin")
	subject2 := &storage.Subject{}
	subject2.SetKind(storage.SubjectKind_SERVICE_ACCOUNT)
	subject2.SetName("another effective admin")
	subject3 := &storage.Subject{}
	subject3.SetKind(storage.SubjectKind_SERVICE_ACCOUNT)
	subject3.SetName("effective admin")
	expected := []*storage.Subject{
		subject,
		subject2,
		subject3,
	}

	protoassert.SlicesEqual(t, expected, getSubjectsGrantedClusterAdmin(inputRoles, inputBindings))
}

func TestFindsRoleswithoutBindings(t *testing.T) {
	inputRoles := []*storage.K8SRole{
		storage.K8SRole_builder{
			Id: "role0",
			Rules: []*storage.PolicyRule{
				storage.PolicyRule_builder{
					ApiGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "create"},
				}.Build(),
			},
		}.Build(),
		storage.K8SRole_builder{
			Id: "role1",
			Rules: []*storage.PolicyRule{
				storage.PolicyRule_builder{
					ApiGroups: []string{""},
					Resources: []string{"pods", "deployments"},
					Verbs:     []string{"get", "list"},
				}.Build(),
			},
		}.Build(),
		storage.K8SRole_builder{
			Id: "role2",
			Rules: []*storage.PolicyRule{
				storage.PolicyRule_builder{
					ApiGroups: []string{""},
					Resources: []string{"deployments"},
					Verbs:     []string{"list"},
				}.Build(),
			},
		}.Build(),
		storage.K8SRole_builder{
			Id: "role3",
			Rules: []*storage.PolicyRule{
				storage.PolicyRule_builder{
					ApiGroups: []string{""},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list"},
				}.Build(),
			},
		}.Build(),
	}
	inputBindings := []*storage.K8SRoleBinding{
		storage.K8SRoleBinding_builder{
			RoleId: "role1",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "robot",
					Namespace: "stackrox",
				}.Build(),
			},
		}.Build(),
		storage.K8SRoleBinding_builder{
			RoleId: "role2",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind:      storage.SubjectKind_USER,
					Name:      "robot",
					Namespace: "stackrox",
				}.Build(),
			},
		}.Build(),
		storage.K8SRoleBinding_builder{
			RoleId: "role3",
			Subjects: []*storage.Subject{
				storage.Subject_builder{
					Kind:      storage.SubjectKind_GROUP,
					Name:      "robot",
					Namespace: "stackrox",
				}.Build(),
			},
		}.Build(),
	}

	inputSubject := &storage.Subject{}
	inputSubject.SetKind(storage.SubjectKind_SERVICE_ACCOUNT)
	inputSubject.SetNamespace("stackrox")
	inputSubject.SetName("robot")

	pr := &storage.PolicyRule{}
	pr.SetApiGroups([]string{""})
	pr.SetResources([]string{"pods", "deployments"})
	pr.SetVerbs([]string{"get", "list"})
	expected := []*storage.PolicyRule{
		pr,
	}

	evaluator := NewEvaluator(inputRoles, inputBindings)
	protoassert.SlicesEqual(t, expected, evaluator.ForSubject(inputSubject).ToSlice())
}
