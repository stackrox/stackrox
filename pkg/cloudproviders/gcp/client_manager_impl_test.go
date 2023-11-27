package gcp

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/cloudproviders/gcp/mocks"
	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/option"
)

// TestClientManager asserts that on the happy path all mutexes are released as expected
// and nothing blocks forever.
func TestClientManager(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)
	mockCredManager := mocks.NewMockCredentialsManager(controller)
	mockCredManager.EXPECT().GetCredentials(gomock.Any()).Return(nil, nil).Times(2)
	mockClientFactory := mocks.NewMockStorageClientFactory(controller)
	var wg sync.WaitGroup
	wg.Add(2)
	mockClientFactory.EXPECT().NewClient(gomock.Any(), gomock.Any()).
		Return(nil, nil).
		Times(2).
		Do(func(context.Context, ...option.ClientOption) { wg.Done() })

	manager := &stsClientManagerImpl{credManager: mockCredManager, storageClientFactory: mockClientFactory}
	manager.updateClients()
	_, done := manager.StorageClient()
	// Simulate an update triggered by a change in the cloud credentials secret while a client is in use.
	go manager.updateClients()
	done()

	wg.Wait()
}
