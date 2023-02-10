package transform

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrongConfigurationTypeTransformRole(t *testing.T) {
	rt := newRoleTransform()
	protos, err := rt.Transform(&declarativeconfig.AuthProvider{})
	assert.Nil(t, protos)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestTransformRole(t *testing.T) {
	role := &declarativeconfig.Role{
		Name:          "some-role",
		Description:   "with a nice description",
		AccessScope:   "and an access scope",
		PermissionSet: "as well as a permission set",
	}

	rt := newRoleTransform()

	protos, err := rt.Transform(role)
	assert.NoError(t, err)

	require.Len(t, protos, 1)
	require.Contains(t, protos, roleType)
	require.Len(t, protos[roleType], 1)

	roleProto, ok := protos[roleType][0].(*storage.Role)
	require.True(t, ok)

	assert.Equal(t, role.Name, roleProto.GetName())
	assert.Equal(t, role.Description, roleProto.GetDescription())
	assert.Equal(t, declarativeconfig.NewDeclarativeAccessScopeUUID(role.AccessScope).String(), roleProto.GetAccessScopeId())
	assert.Equal(t, declarativeconfig.NewDeclarativePermissionSetUUID(role.PermissionSet).String(), roleProto.GetPermissionSetId())
	assert.Equal(t, storage.Traits_DECLARATIVE, roleProto.GetTraits().GetOrigin())
}
