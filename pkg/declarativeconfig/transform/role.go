package transform

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
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

func (r *roleTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]protocompat.Message, error) {
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

	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	roleProto := &storage.Role{}
	roleProto.SetName(roleConfig.Name)
	roleProto.SetDescription(roleConfig.Description)
	roleProto.SetPermissionSetId(stringutils.FirstNonEmpty(accesscontrol.DefaultPermissionSetIDs[roleConfig.PermissionSet],
		declarativeconfig.NewDeclarativePermissionSetUUID(roleConfig.PermissionSet).String()))
	roleProto.SetAccessScopeId(stringutils.FirstNonEmpty(accesscontrol.DefaultAccessScopeIDs[roleConfig.AccessScope],
		declarativeconfig.NewDeclarativeAccessScopeUUID(roleConfig.AccessScope).String()))
	roleProto.SetTraits(traits)
	return map[reflect.Type][]protocompat.Message{
		roleType: {roleProto},
	}, nil
}
