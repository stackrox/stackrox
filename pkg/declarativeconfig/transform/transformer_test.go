package transform

import (
	"testing"

	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	transformer := New()
	castedTransformer, ok := transformer.(*universalTransformer)
	assert.True(t, ok)
	configTransformers := castedTransformer.configurationTransformers

	assert.NotNil(t, configTransformers[declarativeconfig.AccessScopeConfiguration])
	assert.IsType(t, (*accessScopeTransform)(nil), configTransformers[declarativeconfig.AccessScopeConfiguration])

	assert.NotNil(t, configTransformers[declarativeconfig.AuthProviderConfiguration])
	assert.IsType(t, (*authProviderTransform)(nil), configTransformers[declarativeconfig.AuthProviderConfiguration])

	assert.NotNil(t, configTransformers[declarativeconfig.AuthMachineToMachineConfiguration])
	assert.IsType(t, (*authMachineToMachineConfigTransform)(nil), configTransformers[declarativeconfig.AuthMachineToMachineConfiguration])

	assert.NotNil(t, configTransformers[declarativeconfig.PermissionSetConfiguration])
	assert.IsType(t, (*permissionSetTransform)(nil), configTransformers[declarativeconfig.PermissionSetConfiguration])

	assert.NotNil(t, configTransformers[declarativeconfig.RoleConfiguration])
	assert.IsType(t, (*roleTransform)(nil), configTransformers[declarativeconfig.RoleConfiguration])

	assert.NotNil(t, configTransformers[declarativeconfig.NotifierConfiguration])
	assert.IsType(t, (*notifierTransform)(nil), configTransformers[declarativeconfig.NotifierConfiguration])
}
