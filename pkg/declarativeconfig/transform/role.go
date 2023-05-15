package transform

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	_ Transformer = (*roleTransform)(nil)

	roleType = reflect.TypeOf((*storage.Role)(nil))
)

type roleTransform struct{}

func newRoleTransform() *roleTransform {
	return &roleTransform{}
}

func (r *roleTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]proto.Message, error) {
	roleConfig, ok := configuration.(*declarativeconfig.Role)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for role: %T", configuration)
	}

	if roleConfig.Name == "" {
		return nil, errox.InvalidArgs.CausedBy("name must be non-empty")
	}
	if roleConfig.AccessScope == "" {
		return nil, errox.InvalidArgs.CausedBy("access scope must be non-empty")
	}
	if roleConfig.PermissionSet == "" {
		return nil, errox.InvalidArgs.CausedBy("permission set must be non-empty")
	}

	roleProto := &storage.Role{
		Name:        roleConfig.Name,
		Description: roleConfig.Description,
		PermissionSetId: stringutils.FirstNonEmpty(accesscontrol.DefaultPermissionSetIDs[roleConfig.PermissionSet],
			declarativeconfig.NewDeclarativePermissionSetUUID(roleConfig.PermissionSet).String()),
		AccessScopeId: stringutils.FirstNonEmpty(accesscontrol.DefaultAccessScopeIDs[roleConfig.AccessScope],
			declarativeconfig.NewDeclarativeAccessScopeUUID(roleConfig.AccessScope).String()),
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}
	return map[reflect.Type][]proto.Message{
		roleType: {roleProto},
	}, nil
}
