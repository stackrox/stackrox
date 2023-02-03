package transform

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
)

// Transformer transforms a declarativeconfig.Configuration to proto.Message(s).
type Transformer interface {
	Transform(config declarativeconfig.Configuration) ([]proto.Message, error)
}

// New creates a Transformer that can handle transforming all currently supported declarativeconfig.Configuration.
func New() Transformer {
	return &defaultTransformer{configurationTransformers: map[string]Transformer{
		declarativeconfig.AuthProviderConfiguration:  nil,
		declarativeconfig.AccessScopeConfiguration:   nil,
		declarativeconfig.RoleConfiguration:          nil,
		declarativeconfig.PermissionSetConfiguration: nil,
	}}
}

type defaultTransformer struct {
	configurationTransformers map[string]Transformer
}

func (t *defaultTransformer) Transform(config declarativeconfig.Configuration) ([]proto.Message, error) {
	ct, exists := t.configurationTransformers[config.Type()]
	if !exists {
		return nil, errox.NotFound.Newf("no transformation logic for declarative config type %s found", config.Type())
	}
	return ct.Transform(config)
}
