package transform

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
)

var _ Transformer = (*permissionSetTransform)(nil)

var (
	permissionSetType = reflect.TypeOf((*storage.PermissionSet)(nil))
)

type permissionSetTransform struct{}

func newPermissionSetTransform() *permissionSetTransform {
	return &permissionSetTransform{}
}

func (p *permissionSetTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]protocompat.Message, error) {
	permissionSetConfig, ok := configuration.(*declarativeconfig.PermissionSet)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for permission set: %T", configuration)
	}

	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	permissionSetProto := &storage.PermissionSet{}
	permissionSetProto.SetId(declarativeconfig.NewDeclarativePermissionSetUUID(permissionSetConfig.Name).String())
	permissionSetProto.SetName(permissionSetConfig.Name)
	permissionSetProto.SetDescription(permissionSetConfig.Description)
	permissionSetProto.SetResourceToAccess(getResources(permissionSetConfig))
	permissionSetProto.SetTraits(traits)

	return map[reflect.Type][]protocompat.Message{
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
