package rbac

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

type namespacedSubject string

func (ns namespacedSubject) splitNamespaceAndName() (string, string, error) {
	parts := strings.Split(string(ns), "#")
	if len(parts) != 2 {
		return "", "", errors.Errorf("unpacking namespaced subject: expected value to be split by # symbol: %s", string(ns))
	}
	return parts[0], parts[1], nil
}

func nsSubjectFromSubject(s *storage.Subject) namespacedSubject {
	b := strings.Builder{}
	name := s.GetName()
	namespace := s.GetNamespace()
	b.Grow(len(namespace) + len(name) + 1)
	b.WriteString(namespace)
	b.WriteString("#")
	b.WriteString(name)
	return namespacedSubject(b.String())
}

type evaluator struct {
	permissionsForSubject map[namespacedSubject]storage.PermissionLevel
}

func (e *evaluator) GetPermissionLevelForSubject(subject *storage.Subject) storage.PermissionLevel {
	level, ok := e.permissionsForSubject[nsSubjectFromSubject(subject)]
	if !ok {
		return storage.PermissionLevel_NONE
	}
	return level
}

func rolePermissionLevelToClusterPermissionLevel(permissionLevel rolePermissionLevel) storage.PermissionLevel {
	switch permissionLevel {
	case permissionWriteAllResources:
		return storage.PermissionLevel_CLUSTER_ADMIN
	case permissionWriteSomeResource, permissionListSomeResource, permissionGetOrWatchSomeResource:
		return storage.PermissionLevel_ELEVATED_CLUSTER_WIDE
	case permissionNone:
		return storage.PermissionLevel_NONE
	}
	utils.Should(fmt.Errorf("unhandled permission level %d", permissionLevel))
	return storage.PermissionLevel_UNSET
}

func rolePermissionLevelToNamespacePermissionLevel(permissionLevel rolePermissionLevel) storage.PermissionLevel {
	switch permissionLevel {
	case permissionWriteAllResources, permissionWriteSomeResource, permissionListSomeResource:
		return storage.PermissionLevel_ELEVATED_IN_NAMESPACE
	case permissionGetOrWatchSomeResource:
		return storage.PermissionLevel_DEFAULT
	case permissionNone:
		return storage.PermissionLevel_NONE
	}
	utils.Should(fmt.Errorf("unhandled permission level %d", permissionLevel))
	return storage.PermissionLevel_UNSET
}

func newBucketEvaluator(roles map[namespacedRoleRef]namespacedRole, bindings map[namespacedBindingID]*namespacedBinding) *evaluator {
	permissionsForSubject := make(map[namespacedSubject]storage.PermissionLevel, len(bindings))

	for bID, b := range bindings {
		role, ok := roles[b.roleRef]
		if !ok {
			continue // This roleRef is dangling, no rules for us to use.
		}

		for _, subject := range b.subjects {
			currentLevel, ok := permissionsForSubject[subject]
			if !ok {
				permissionsForSubject[subject] = storage.PermissionLevel_NONE
				currentLevel = storage.PermissionLevel_NONE
			}

			var roleLevel storage.PermissionLevel
			if bID.IsClusterBinding() {
				roleLevel = rolePermissionLevelToClusterPermissionLevel(role.permissionLevel)
			} else {
				roleLevel = rolePermissionLevelToNamespacePermissionLevel(role.permissionLevel)
			}

			if roleLevel > currentLevel {
				permissionsForSubject[subject] = roleLevel
			}
		}
	}
	return &evaluator{permissionsForSubject: permissionsForSubject}
}
