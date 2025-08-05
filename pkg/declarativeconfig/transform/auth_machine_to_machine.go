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
		mappings = append(mappings, &storage.AuthMachineToMachineConfig_Mapping{
			Key:             mapping.Key,
			ValueExpression: mapping.ValueExpression,
			Role:            mapping.Role,
		})
	}
	m2mAuthConfigProto := &storage.AuthMachineToMachineConfig{
		Id:                      declarativeconfig.NewDeclarativeM2MAuthConfigUUID(authM2MConfig.Issuer).String(),
		Type:                    storage.AuthMachineToMachineConfig_Type(authM2MConfig.Type),
		TokenExpirationDuration: authM2MConfig.TokenExpirationDuration,
		Mappings:                mappings,
		Issuer:                  authM2MConfig.Issuer,
		Traits:                  &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}

	return map[reflect.Type][]protocompat.Message{
		authM2MConfigType: {m2mAuthConfigProto},
	}, nil
}
