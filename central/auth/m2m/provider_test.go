package m2m

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewProvider(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	cfgId := uuid.NewTestUUID(1).String()
	configType := storage.AuthMachineToMachineConfig_GENERIC
	mockRoleMapper := permissionsMocks.NewMockRoleMapper(mockCtrl)
	provider := newProviderFromConfig(cfgId, configType.String(), mockRoleMapper)
	assert.NotNil(t, provider)
	assert.Equal(t, cfgId, provider.ID())
	providerName := fmt.Sprintf("%s-%s", configType.String(), cfgId)
	assert.Equal(t, providerName, provider.Name())
	assert.Equal(t, configType.String(), provider.Type())
	assert.Equal(t, mockRoleMapper, provider.RoleMapper())
}
