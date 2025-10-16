package transform

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
)

var (
	_ Transformer = (*authMachineToMachineConfigTransform)(nil)

	authM2MConfigType = reflect.TypeOf((*storage.AuthMachineToMachineConfig)(nil))
)

type authMachineToMachineConfigTransform struct{}

func newAuthMachineToMachineConfigTransform() *authMachineToMachineConfigTransform {
	return &authMachineToMachineConfigTransform{}
}

func (t authMachineToMachineConfigTransform) Transform(
	configuration declarativeconfig.Configuration,
) (map[reflect.Type][]protocompat.Message, error) {
	authM2MConfig, ok := configuration.(*declarativeconfig.AuthMachineToMachineConfig)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for machine to machine auth configuration: %T", configuration)
	}
	mappings := make([]*storage.AuthMachineToMachineConfig_Mapping, 0, len(authM2MConfig.Mappings))
	for _, mapping := range authM2MConfig.Mappings {
		am := &storage.AuthMachineToMachineConfig_Mapping{}
		am.SetKey(mapping.Key)
		am.SetValueExpression(mapping.ValueExpression)
		am.SetRole(mapping.Role)
		mappings = append(mappings, am)
	}
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	m2mAuthConfigProto := &storage.AuthMachineToMachineConfig{}
	m2mAuthConfigProto.SetId(declarativeconfig.NewDeclarativeM2MAuthConfigUUID(authM2MConfig.Issuer).String())
	m2mAuthConfigProto.SetType(storage.AuthMachineToMachineConfig_Type(authM2MConfig.Type))
	m2mAuthConfigProto.SetTokenExpirationDuration(authM2MConfig.TokenExpirationDuration)
	m2mAuthConfigProto.SetMappings(mappings)
	m2mAuthConfigProto.SetIssuer(authM2MConfig.Issuer)
	m2mAuthConfigProto.SetTraits(traits)

	return map[reflect.Type][]protocompat.Message{
		authM2MConfigType: {m2mAuthConfigProto},
	}, nil
}
