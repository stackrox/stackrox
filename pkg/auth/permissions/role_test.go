package permissions

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleHasPermissions(t *testing.T) {
	type expectation struct {
		view   bool
		modify bool
	}
	writeAccessibleResource := Resource("writeaccessible")
	readAccessibleResource := Resource("readaccessible")
	forbiddenResource := Resource("forbidden")

	role := NewRoleWithPermissions("Testrole", Modify(writeAccessibleResource), View(readAccessibleResource))

	expectations := map[Resource]expectation{
		writeAccessibleResource: {view: true, modify: true},
		readAccessibleResource:  {view: true},
		forbiddenResource:       {},
	}

	for resource, exp := range expectations {
		t.Run(fmt.Sprintf("resource: %s", resource), func(t *testing.T) {
			assert.Equal(t, exp.view, RoleHasPermission(role, View(resource)))
			assert.Equal(t, exp.modify, RoleHasPermission(role, Modify(resource)))
		})
	}
}
