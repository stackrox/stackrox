package transform

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrongConfigurationTypeTransformPermissionSet(t *testing.T) {
	pt := newPermissionSetTransform()
	protos, err := pt.Transform(&declarativeconfig.AuthProvider{})
	assert.Nil(t, protos)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestTransformPermissionSet(t *testing.T) {
	permissionSet := &declarativeconfig.PermissionSet{
		Name:        "some-permission-set",
		Description: "with a nice description",
		Resources: []declarativeconfig.ResourceWithAccess{
			{Resource: "some-resource", Access: declarativeconfig.Access(storage.Access_NO_ACCESS)},
			{Resource: "another-resource", Access: declarativeconfig.Access(storage.Access_READ_ACCESS)},
			{Resource: "yet-another-resource", Access: declarativeconfig.Access(storage.Access_READ_WRITE_ACCESS)},
		},
	}
	expectedPermissionSetID := declarativeconfig.NewDeclarativePermissionSetUUID(permissionSet.Name).String()
	expectedResourceToAccess := map[string]storage.Access{
		"some-resource":        storage.Access_NO_ACCESS,
		"another-resource":     storage.Access_READ_ACCESS,
		"yet-another-resource": storage.Access_READ_WRITE_ACCESS,
	}

	pt := newPermissionSetTransform()

	protos, err := pt.Transform(permissionSet)
	assert.NoError(t, err)

	require.Len(t, protos, 1)
	require.Contains(t, protos, permissionSetType)
	require.Len(t, protos[permissionSetType], 1)

	permissionSetProto, ok := protos[permissionSetType][0].(*storage.PermissionSet)
	require.True(t, ok)

	assert.Equal(t, expectedPermissionSetID, permissionSetProto.GetId())
	assert.Equal(t, permissionSet.Name, permissionSetProto.GetName())
	assert.Equal(t, permissionSet.Description, permissionSetProto.GetDescription())
	assert.Equal(t, expectedResourceToAccess, permissionSetProto.GetResourceToAccess())
	assert.Equal(t, storage.Traits_DECLARATIVE, permissionSetProto.GetTraits().GetOrigin())
}

func TestUniversalTransformPermissionSet(t *testing.T) {
	permissionSet := &declarativeconfig.PermissionSet{
		Name:        "some-permission-set",
		Description: "with a nice description",
		Resources: []declarativeconfig.ResourceWithAccess{
			{Resource: "some-resource", Access: declarativeconfig.Access(storage.Access_NO_ACCESS)},
			{Resource: "another-resource", Access: declarativeconfig.Access(storage.Access_READ_ACCESS)},
			{Resource: "yet-another-resource", Access: declarativeconfig.Access(storage.Access_READ_WRITE_ACCESS)},
		},
	}
	expectedPermissionSetID := declarativeconfig.NewDeclarativePermissionSetUUID(permissionSet.Name).String()
	expectedResourceToAccess := map[string]storage.Access{
		"some-resource":        storage.Access_NO_ACCESS,
		"another-resource":     storage.Access_READ_ACCESS,
		"yet-another-resource": storage.Access_READ_WRITE_ACCESS,
	}

	ut := New()

	protos, err := ut.Transform(permissionSet)
	assert.NoError(t, err)

	require.Len(t, protos, 1)
	require.Contains(t, protos, permissionSetType)
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ps := &storage.PermissionSet{}
	ps.SetId(expectedPermissionSetID)
	ps.SetName(permissionSet.Name)
	ps.SetDescription(permissionSet.Description)
	ps.SetResourceToAccess(expectedResourceToAccess)
	ps.SetTraits(traits)
	expectedMessages := []*storage.PermissionSet{
		ps,
	}

	obtainedMessages := make([]*storage.PermissionSet, 0, len(protos[permissionSetType]))
	for _, m := range protos[permissionSetType] {
		casted, ok := m.(*storage.PermissionSet)
		if ok {
			obtainedMessages = append(obtainedMessages, casted)
		}
	}
	protoassert.SlicesEqual(t, expectedMessages, obtainedMessages)
}
