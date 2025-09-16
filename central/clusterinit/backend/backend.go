package backend

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/backend/certificate"
	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/grpc/authn"
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

// CRSWithMeta contains a CRS alongside its meta data.
type CRSWithMeta struct {
	CRS  *crs.CRS
	Meta *storage.InitBundleMeta
}

var (
	// ErrInitBundleIsRevoked signals that an init bundle has been revoked.
	ErrInitBundleIsRevoked = errors.New("init bundle is revoked")
	// ErrCRSIsRevoked signals that a CRS has been revoked.
	ErrCRSIsRevoked = errors.New("CRS is revoked")
)

// Backend is the backend for the cluster-init component.
//
//go:generate mockgen-wrapper
type Backend interface {
	GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error)
	GetAllCRS(ctx context.Context) ([]*storage.InitBundleMeta, error)
	GetCAConfig(ctx context.Context) (*CAConfig, error)
	Issue(ctx context.Context, name string) (*InitBundleWithMeta, error)
	IssueCRS(ctx context.Context, name string, validUntil time.Time) (*CRSWithMeta, error)
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
