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
	config := &storage.AuthMachineToMachineConfig{
		Id:   cfgId,
		Type: storage.AuthMachineToMachineConfig_GENERIC,
	}
	mockRoleMapper := permissionsMocks.NewMockRoleMapper(mockCtrl)
	provider := newProviderFromConfig(config, mockRoleMapper)
	assert.NotNil(t, provider)
	assert.Equal(t, cfgId, provider.ID())
	providerName := fmt.Sprintf("%s-%s", storage.AuthMachineToMachineConfig_GENERIC.String(), cfgId)
	assert.Equal(t, providerName, provider.Name())
	assert.Equal(t, storage.AuthMachineToMachineConfig_GENERIC.String(), provider.Type())
	assert.Equal(t, mockRoleMapper, provider.RoleMapper())
}
