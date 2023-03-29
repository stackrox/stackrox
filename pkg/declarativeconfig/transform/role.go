package transform

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
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

	roleProto := &storage.Role{
		Name:            roleConfig.Name,
		Description:     roleConfig.Description,
		PermissionSetId: declarativeconfig.NewDeclarativePermissionSetUUID(roleConfig.PermissionSet).String(),
		AccessScopeId:   declarativeconfig.NewDeclarativeAccessScopeUUID(roleConfig.AccessScope).String(),
		Traits:          &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}
	return map[reflect.Type][]proto.Message{
		roleType: {roleProto},
	}, nil
}
