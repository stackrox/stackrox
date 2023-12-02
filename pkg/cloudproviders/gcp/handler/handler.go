package handler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/types"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/oauth2/google"
)

type Handler[T types.GcpSDKClients] interface {
	UpdateClient(ctx context.Context, creds *google.Credentials) error
	GetClient() (T, types.DoneFunc)
}

type handlerImpl[T types.GcpSDKClients] struct {
	client  T
	mutex   sync.Mutex
	wg      *concurrency.WaitGroup
	factory ClientFactory[T]
}

func (h *handlerImpl[T]) UpdateClient(ctx context.Context, creds *google.Credentials) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if concurrency.WaitInContext(h.wg, ctx) {
		return ctx.Err()
	}

	client, err := h.factory.NewClient(ctx, creds)
	if err != nil {
		return errors.Wrap(err, "failed to create client")
	}
	h.client = client
	return nil
}

func (h *handlerImpl[T]) GetClient() (T, types.DoneFunc) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.wg.Add(1)
	return h.client, func() { h.wg.Add(-1) }
}

// NewHandlerNoInit creates a handler without initializing credentials.
func NewHandlerNoInit[T types.GcpSDKClients]() Handler[T] {
	wg := concurrency.NewWaitGroup(0)
	return &handlerImpl[T]{factory: GetClientFactory(*new(T)), wg: &wg}
}

// NewHandler creates a handler initialized with the given credentials.
func NewHandler[T types.GcpSDKClients](ctx context.Context, creds *google.Credentials) (Handler[T], error) {
	wg := concurrency.NewWaitGroup(0)
	h := &handlerImpl[T]{factory: GetClientFactory(*new(T)), wg: &wg}
	if err := h.UpdateClient(ctx, creds); err != nil {
		return nil, errors.Wrap(err, "updating client")
	}
	return h, nil
}
