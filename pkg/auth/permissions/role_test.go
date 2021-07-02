package permissions

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestRoleNewUnionPermissions(t *testing.T) {
	resolvedRole1 := &ResolvedRole{
		PermissionSet: &storage.PermissionSet{
			ResourceToAccess: map[string]storage.Access{
				"A": storage.Access_READ_ACCESS,
				"B": storage.Access_READ_ACCESS,
			},
		},
	}
	resolvedRole2 := &ResolvedRole{
		PermissionSet: &storage.PermissionSet{
			ResourceToAccess: map[string]storage.Access{
				"B": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	// for single role we just return role's resourceToAccess
	union1 := NewUnionPermissions([]*ResolvedRole{resolvedRole1})
	expected1 := &storage.ResourceToAccess{
		ResourceToAccess: resolvedRole1.GetResourceToAccess(),
	}
	assert.Equal(t, expected1, union1)

	union2 := NewUnionPermissions([]*ResolvedRole{resolvedRole1, resolvedRole2})
	expected2 := &storage.ResourceToAccess{
		ResourceToAccess: map[string]storage.Access{
			"A": storage.Access_READ_ACCESS,
			"B": storage.Access_READ_WRITE_ACCESS,
		},
	}
	assert.Equal(t, expected2, union2)
}
