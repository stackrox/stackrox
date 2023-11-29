package auth

import (
	"testing"

	authMocks "github.com/stackrox/rox/pkg/cloudproviders/gcp/auth/mocks"
	storageMocks "github.com/stackrox/rox/pkg/cloudproviders/gcp/storage/mocks"
	"go.uber.org/mock/gomock"
)

// TestClientManager asserts that on the happy path the factory update is called.
func TestClientManager(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)

	mockCredManager := authMocks.NewMockCredentialsManager(controller)
	mockCredManager.EXPECT().GetCredentials(gomock.Any()).Return(nil, nil)
	mockClientFactory := storageMocks.NewMockClientHandler(controller)
	mockClientFactory.EXPECT().UpdateClient(gomock.Any(), gomock.Any()).Return(nil)

	manager := &stsClientManagerImpl{credManager: mockCredManager, storageClientHandler: mockClientFactory}
	manager.updateClients()
}
