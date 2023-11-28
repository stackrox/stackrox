package storage

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/oauth2/google"
)

// ClientHandler handles a GCP storage client.
//
//go:generate mockgen-wrapper
type ClientHandler interface {
	UpdateClient(ctx context.Context, creds *google.Credentials) error
	GetClient() (*storage.Client, func())
}

type clientHandlerImpl struct {
	factory ClientFactory
	client  *storage.Client
	mutex   sync.Mutex
	wg      sync.WaitGroup
}

var _ ClientHandler = &clientHandlerImpl{}

// NewClientHandlerNoInit creates a new storage client handler without initializing the client.
//
// Not initializing the client is useful to start the cloud credential secret controller in
// environments where no valid GCP credential can be constructed. In these cases, we must
// leave the client nil until a cloud credential secret is added.
func NewClientHandlerNoInit() ClientHandler {
	return &clientHandlerImpl{factory: &clientFactoryImpl{}}
}

// NewClientHandler creates a new storage client handler.
func NewClientHandler(ctx context.Context, creds *google.Credentials) (ClientHandler, error) {
	handler := &clientHandlerImpl{factory: &clientFactoryImpl{}}
	if err := handler.UpdateClient(ctx, creds); err != nil {
		return nil, errors.Wrap(err, "updating client")
	}
	return handler, nil
}

func (s *clientHandlerImpl) UpdateClient(ctx context.Context, creds *google.Credentials) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.wg.Wait()

	client, err := s.factory.NewClient(ctx, creds)
	if err != nil {
		return errors.Wrap(err, "failed to create GCP storage client")
	}
	s.client = client
	return nil
}

func (s *clientHandlerImpl) GetClient() (*storage.Client, func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.wg.Add(1)
	return s.client, func() { s.wg.Done() }
}
