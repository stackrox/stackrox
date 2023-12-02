package handler

import (
	"context"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/handler/mocks"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2/google"
)

func TestClientHandler(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	wg.Add(1)

	mockClientFactory := mocks.NewMockClientFactory[*storage.Client](gomock.NewController(t))
	wgHandler := concurrency.NewWaitGroup(0)
	h := &handlerImpl[*storage.Client]{factory: mockClientFactory, wg: &wgHandler}

	ctx := context.Background()
	mockClientFactory.EXPECT().NewClient(ctx, nil).Return(&storage.Client{}, nil).
		Do(func(context.Context, *google.Credentials) { wg.Done() })

	_, done := h.GetClient()

	go func() {
		require.NoError(t, h.UpdateClient(ctx, nil))
	}()
	done()
	wg.Wait()
}
