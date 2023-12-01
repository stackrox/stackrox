package storage

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/cloudproviders/gcp/storage/mocks"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2/google"
)

// TestClientHandler asserts that on the happy path all mutexes are released as expected
// and nothing blocks forever.
func TestClientHandler(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	wg.Add(1)

	controller := gomock.NewController(t)
	mockClientFactory := mocks.NewMockClientFactory(controller)
	wgHandler := concurrency.NewWaitGroup(0)
	handler := &clientHandlerImpl{factory: mockClientFactory, wg: &wgHandler}
	ctx := context.Background()
	mockClientFactory.EXPECT().NewClient(ctx, nil).
		Return(nil, nil).
		Do(func(context.Context, *google.Credentials) { wg.Done() })

	_, done := handler.GetClient()
	go func() {
		err := handler.UpdateClient(ctx, nil)
		require.NoError(t, err)
	}()
	done()
	wg.Wait()
}
