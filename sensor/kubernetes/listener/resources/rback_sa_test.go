package resources

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRBACUpdaterPermissionLevelForSubject(t *testing.T) {
	// This test creates roles and bindings and then updates them to match following state:
	// Roles:
	//  1. role-admin (all verbs on all resources)
	//  2. role-default (get)
	// Bindings:
	//  1. admin-subject -> role-admin
	//  2. default-subject -> role-default
	// Cluster Roles:
	//  1. cluster-admin (all verbs on all resources)
	//  2. cluster-elevated (get on all resources)
	// Cluster Bindings:
	//  1. cluster-admin-subject -> cluster-admin
	//  2. cluster-elevated-subject -> cluster-elevated
	roles := []*v1.Role{
		{
			ObjectMeta: meta("role-admin"),
		},
		{
			ObjectMeta: meta("role-default"),
		},
		{
			ObjectMeta: meta("role-admin"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		},
		{
			ObjectMeta: meta("role-default"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"get"},
			}},
		},
	}
	bindings := []*v1.RoleBinding{
		{
			ObjectMeta: meta("b1"),
			RoleRef:    role("role-admin"),
		},
		{
			ObjectMeta: meta("b2"),
			RoleRef:    role("role-default"),
		},
		{
			ObjectMeta: meta("b1"),
			RoleRef:    role("role-admin"),
			Subjects: []v1.Subject{
				{
					Name:      "admin-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
		{
			ObjectMeta: meta("b2"),
			RoleRef:    role("role-default"),
			Subjects: []v1.Subject{{
				Name:      "default-subject",
				Kind:      v1.ServiceAccountKind,
				Namespace: "n1",
			}},
		},
	}
	clusterRoles := []*v1.ClusterRole{
		{
			ObjectMeta: meta("cluster-admin"),
		},
		{
			ObjectMeta: meta("cluster-elevated"),
		},
		{
			ObjectMeta: meta("cluster-admin"),

			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			}},
		},
	}
	clusterBindings := []*v1.ClusterRoleBinding{
		{
			ObjectMeta: meta("b3"),
			RoleRef:    clusterRole("cluster-admin"),
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
		},
		{
			ObjectMeta: meta("b3"),
			RoleRef:    clusterRole("cluster-admin"),
			Subjects: []v1.Subject{{
				Name: "cluster-admin-subject",
				Kind: v1.ServiceAccountKind,
			}},
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
			Subjects: []v1.Subject{
				{
					Name:      "cluster-elevated-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-elevated-subject-2",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
	}

	testCases := []struct {
		subject, namespace string
		expected           storage.PermissionLevel
	}{
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, subject: "cluster-elevated-subject", namespace: "n1"},
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, subject: "cluster-elevated-subject-2", namespace: "n1"},
		{expected: storage.PermissionLevel_NONE, subject: "cluster-elevated-subject"},
		{expected: storage.PermissionLevel_NONE, subject: "cluster-admin-subject", namespace: "n1"},
		{expected: storage.PermissionLevel_CLUSTER_ADMIN, subject: "cluster-admin-subject"},
		{expected: storage.PermissionLevel_ELEVATED_IN_NAMESPACE, subject: "admin-subject", namespace: "n1"},
		{expected: storage.PermissionLevel_DEFAULT, subject: "default-subject", namespace: "n1"},
		{expected: storage.PermissionLevel_NONE, subject: "default-subject"},
		{expected: storage.PermissionLevel_NONE, subject: "admin-subject"},
	}
	for _, synced := range []bool{true, false} {
		updater := setupUpdater(roles, clusterRoles, bindings, clusterBindings, synced)
		updaterWithNoRoles := setupUpdater(roles, clusterRoles, bindings, clusterBindings, synced)
		for _, r := range roles {
			updaterWithNoRoles.removeRole(r)
		}
		for _, r := range clusterRoles {
			updaterWithNoRoles.removeClusterRole(r)
		}
		updaterWithNoBindings := setupUpdater(roles, clusterRoles, bindings, clusterBindings, synced)
		for _, b := range bindings {
			updaterWithNoBindings.removeBinding(b)
		}
		for _, b := range clusterBindings {
			updaterWithNoBindings.removeClusterBinding(b)
		}
		for _, tc := range testCases {
			tc := tc

			name := fmt.Sprintf("%s in namespace %q should have %s permision level, synced: %t", tc.subject, tc.namespace, tc.expected, synced)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tc.expected, permissionLevelForSubjectInNamespace(updater, tc.subject, tc.namespace))
			})

			name = fmt.Sprintf("%s in namespace %q should have NO permisions after removing roles but keeping bindings, synced: %t", tc.subject, tc.namespace, synced)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, storage.PermissionLevel_NONE, permissionLevelForSubjectInNamespace(updaterWithNoRoles, tc.subject, tc.namespace))
			})

			name = fmt.Sprintf("%s in namespace %q should have NO permisions after removing bindings but keeping roles, synced: %t", tc.subject, tc.namespace, synced)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, storage.PermissionLevel_NONE, permissionLevelForSubjectInNamespace(updaterWithNoBindings, tc.subject, tc.namespace))
			})
		}
	}
}

func role(name string) v1.RoleRef {
	return roleRef(name, "Role")
}

func clusterRole(name string) v1.RoleRef {
	return roleRef(name, "ClusterRole")
}

func roleRef(name, kind string) v1.RoleRef {
	return v1.RoleRef{
		Name: name, Kind: kind, APIGroup: "rbac.authorization.k8s.io",
	}
}

func meta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: name, UID: types.UID(name + "-id"), Namespace: "n1",
	}
}

func setupUpdater(roles []*v1.Role, clusterRoles []*v1.ClusterRole, bindings []*v1.RoleBinding, clusterBindings []*v1.ClusterRoleBinding, synced bool) rbacUpdater {
	var flagInitialRbacLoadDone concurrency.Flag
	tested := newRBACUpdater(&flagInitialRbacLoadDone)
	flagInitialRbacLoadDone.Set(synced)
	for _, r := range roles {
		tested.upsertRole(r)
	}
	for _, b := range bindings {
		tested.upsertBinding(b)
	}
	for _, r := range clusterRoles {
		tested.upsertClusterRole(r)
	}
	for _, b := range clusterBindings {
		tested.upsertClusterBinding(b)
	}
	flagInitialRbacLoadDone.Set(true)
	return tested
}

func permissionLevelForSubjectInNamespace(updater rbacUpdater, role, ns string) storage.PermissionLevel {
	deployment := deploymentWrap{Deployment: &storage.Deployment{
		ServiceAccount: role,
		Namespace:      ns,
	}}
	updater.assignPermissionLevelToDeployment(&deployment)
	return deployment.ServiceAccountPermissionLevel
}
