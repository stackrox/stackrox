package auth

import (
	"testing"

	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	"cloud.google.com/go/storage"
	authMocks "github.com/stackrox/rox/pkg/cloudproviders/gcp/auth/mocks"
	handlerMocks "github.com/stackrox/rox/pkg/cloudproviders/gcp/handler/mocks"
	"go.uber.org/mock/gomock"
)

// TestClientManager asserts that on the happy path the factory update is called.
func TestClientManager(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)

	mockCredManager := authMocks.NewMockCredentialsManager(controller)
	mockCredManager.EXPECT().GetCredentials(gomock.Any()).Return(nil, nil)
	mockStorageHandler := handlerMocks.NewMockHandler[*storage.Client](controller)
	mockStorageHandler.EXPECT().UpdateClient(gomock.Any(), gomock.Any()).Return(nil)
	mockSecurityCenterHandler := handlerMocks.NewMockHandler[*securitycenter.Client](controller)
	mockSecurityCenterHandler.EXPECT().UpdateClient(gomock.Any(), gomock.Any()).Return(nil)

	manager := &stsClientManagerImpl{
		credManager:                 mockCredManager,
		storageClientHandler:        mockStorageHandler,
		securityCenterClientHandler: mockSecurityCenterHandler,
	}
	manager.updateClients()
}
