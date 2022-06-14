package backend

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/clusterinit/backend/certificate"
	"github.com/stackrox/stackrox/central/clusterinit/store"
	"github.com/stackrox/stackrox/central/clusters"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/grpc/authn"
)

// CAConfig is the configuration for the StackRox Service CA.
type CAConfig struct {
	CACert string
}

// InitBundleWithMeta contains an init bundle alongside its meta data.
type InitBundleWithMeta struct {
	CAConfig
	CertBundle clusters.CertBundle
	Meta       *storage.InitBundleMeta
}

var (
	// ErrInitBundleIsRevoked signals that an init bundle has been revoked.
	ErrInitBundleIsRevoked = errors.New("init bundle is revoked")
)

// Backend is the backend for the cluster-init component.
//go:generate mockgen-wrapper
type Backend interface {
	GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error)
	GetCAConfig(ctx context.Context) (*CAConfig, error)
	Issue(ctx context.Context, name string) (*InitBundleWithMeta, error)
	Revoke(ctx context.Context, id string) error
	CheckRevoked(ctx context.Context, id string) error
	authn.ValidateCertChain
}

func newBackend(store store.Store, certProvider certificate.Provider) Backend {
	return &backendImpl{
		store:        store,
		certProvider: certProvider,
	}
}
