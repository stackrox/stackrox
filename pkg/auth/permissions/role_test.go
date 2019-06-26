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
