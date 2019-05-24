package manager

import (
	"context"

	"github.com/stackrox/rox/central/clientca/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sync"
)

// ClientCAManager is the manager interface for client CA certificates
type ClientCAManager interface {
	GetClientCA(ctx context.Context, id string) (*storage.Certificate, bool)
	GetAllClientCAs(ctx context.Context) []*storage.Certificate
	AddClientCA(ctx context.Context, certificatePEM string) (*storage.Certificate, error)
	RemoveClientCA(ctx context.Context, id string) error
	TLSConfigurer() verifier.TLSConfigurer
	Initialize() error
}

// New returns a ClientCAManager. You should call Initialize on it before proceeding.
func New(store store.Store) ClientCAManager {
	return &managerImpl{
		store:    store,
		mutex:    sync.RWMutex{},
		allCerts: nil,
	}
}
