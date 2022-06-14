package analysis

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stretchr/testify/assert"
)

func TestFindsRedundantRoleBindings(t *testing.T) {
	inputBindings := []*storage.K8SRoleBinding{
		{
			RoleId: "role",
			Labels: defaultLabelMap, // Default binding, should be ignored
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      k8srbac.DefaultServiceAccountName,
					Namespace: "ns1",
				},
			},
		},
		{
			RoleId: "role", // Matches first, but isn't default, so neither should be returned.
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      k8srbac.DefaultServiceAccountName,
					Namespace: "ns1",
				},
			},
		},
		{
			RoleId: "role", // Equal to binding below
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_USER,
					Name:      "John",
					Namespace: "ns1",
				},
			},
		},
		{
			RoleId: "role", // Equal to binding above
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_USER,
					Name:      "John",
					Namespace: "ns1",
				},
			},
		},
		{
			RoleId: "role", // Shadows both bindings above.
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_USER,
					Name:      "John",
					Namespace: "ns1",
				},
				{
					Kind:      storage.SubjectKind_USER,
					Name:      "Steve",
					Namespace: "ns1",
				},
			},
		},
		{
			RoleId: "role", // Different namespaces for subjects, so should be ignored.
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_USER,
					Name:      "John",
					Namespace: "ns2",
				},
				{
					Kind:      storage.SubjectKind_USER,
					Name:      "Steve",
					Namespace: "ns2",
				},
			},
		},
	}
	expected := map[*storage.K8SRoleBinding]*MatchingRoleBindings{
		inputBindings[2]: {
			Equivalent: []*storage.K8SRoleBinding{
				inputBindings[3],
			},
		},
		inputBindings[3]: {
			Equivalent: []*storage.K8SRoleBinding{
				inputBindings[2],
			},
		},
		inputBindings[4]: {
			Shadows: []*storage.K8SRoleBinding{
				inputBindings[2],
				inputBindings[3],
			},
		},
	}

	assert.Equal(t, expected, getRedundantRoleBindings(inputBindings))
}
