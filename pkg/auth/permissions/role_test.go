package permissions

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestRoleHasPermissions(t *testing.T) {
	type expectation struct {
		view   bool
		modify bool
	}
	writeAccessibleResource := ResourceMetadata{
		Resource: Resource("writeaccessible"),
	}
	readAccessibleResource := ResourceMetadata{
		Resource: Resource("readaccessible"),
	}
	forbiddenResource := ResourceMetadata{
		Resource: Resource("forbidden"),
	}

	role := NewRoleWithAccess("Testrole", Modify(writeAccessibleResource), View(readAccessibleResource))

	expectations := map[ResourceMetadata]expectation{
		writeAccessibleResource: {view: true, modify: true},
		readAccessibleResource:  {view: true},
		forbiddenResource:       {},
	}

	for resourceMetadata, exp := range expectations {
		t.Run(fmt.Sprintf("resource: %s", resourceMetadata), func(t *testing.T) {
			assert.Equal(t, exp.view, RoleHasPermission(role, View(resourceMetadata)))
			assert.Equal(t, exp.modify, RoleHasPermission(role, Modify(resourceMetadata)))
		})
	}
}

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
