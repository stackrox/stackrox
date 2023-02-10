package transform

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
)

var _ Transformer = (*permissionSetTransform)(nil)

var (
	permissionSetType = reflect.TypeOf((*storage.PermissionSet)(nil))
)

type permissionSetTransform struct{}

func newPermissionSetTransform() *permissionSetTransform {
	return &permissionSetTransform{}
}

func (p *permissionSetTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]proto.Message, error) {
	permissionSetConfig, ok := configuration.(*declarativeconfig.PermissionSet)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for permission set: %T", configuration)
	}

	permissionSetProto := &storage.PermissionSet{
		Id:               declarativeconfig.NewDeclarativePermissionSetUUID(permissionSetConfig.Name).String(),
		Name:             permissionSetConfig.Name,
		Description:      permissionSetConfig.Description,
		ResourceToAccess: getResources(permissionSetConfig),
		Traits:           &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}

	return map[reflect.Type][]proto.Message{
		permissionSetType: {permissionSetProto},
	}, nil
}

func getResources(permissionSetConfig *declarativeconfig.PermissionSet) map[string]storage.Access {
	resourceToAccess := make(map[string]storage.Access, len(permissionSetConfig.Resources))
	for _, resource := range permissionSetConfig.Resources {
		resourceToAccess[resource.Resource] = storage.Access(resource.Access)
	}
	return resourceToAccess
}
