package utils

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
)

func TestRoleNewUnionPermissions(t *testing.T) {
	resolvedRole1 := roletest.NewResolvedRoleWithDenyAll(
		"role1",
		map[string]storage.Access{
			"A": storage.Access_READ_ACCESS,
			"B": storage.Access_READ_ACCESS,
		})
	resolvedRole2 := roletest.NewResolvedRoleWithDenyAll(
		"role2",
		map[string]storage.Access{
			"B": storage.Access_READ_WRITE_ACCESS,
		})

	// For single role we just return role's resourceToAccess.
	union1 := NewUnionPermissions([]permissions.ResolvedRole{resolvedRole1})
	expected1 := resolvedRole1.GetPermissions()
	assert.Equal(t, expected1, union1)

	union2 := NewUnionPermissions([]permissions.ResolvedRole{resolvedRole1, resolvedRole2})
	expected2 := map[string]storage.Access{
		"A": storage.Access_READ_ACCESS,
		"B": storage.Access_READ_WRITE_ACCESS,
	}
	assert.Equal(t, expected2, union2)
}
