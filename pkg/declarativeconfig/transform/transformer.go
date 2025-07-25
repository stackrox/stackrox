package transform

import (
	"reflect"

	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
)

var _ Transformer = (*universalTransformer)(nil)

// Transformer transforms a declarativeconfig.Configuration to protocompat.Message(s).
//
//go:generate mockgen-wrapper
type Transformer interface {
	Transform(config declarativeconfig.Configuration) (map[reflect.Type][]protocompat.Message, error)
}

// New creates a Transformer that can handle transforming all currently supported declarativeconfig.Configuration.
func New() Transformer {
	return &universalTransformer{configurationTransformers: map[string]Transformer{
		declarativeconfig.AccessScopeConfiguration:          newAccessScopeTransform(),
		declarativeconfig.AuthMachineToMachineConfiguration: newAuthMachineToMachineConfigTransform(),
		declarativeconfig.AuthProviderConfiguration:         newAuthProviderTransformer(),
		declarativeconfig.NotifierConfiguration:             newNotifierTransform(),
		declarativeconfig.PermissionSetConfiguration:        newPermissionSetTransform(),
		declarativeconfig.RoleConfiguration:                 newRoleTransform(),
	}}
}

type universalTransformer struct {
	configurationTransformers map[string]Transformer
}

func (t *universalTransformer) Transform(config declarativeconfig.Configuration) (map[reflect.Type][]protocompat.Message, error) {
	ct, exists := t.configurationTransformers[config.ConfigurationType()]
	if !exists {
		return nil, errox.InvariantViolation.Newf("no transformation logic for declarative config type %s found",
			config.ConfigurationType())
	}
	return ct.Transform(config)
}
