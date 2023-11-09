package backend

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/backend/access"
	"github.com/stackrox/rox/central/clusterinit/backend/certificate"
	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cache/storebased"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
)

var _ authn.ValidateCertChain = (*backendImpl)(nil)

type backendImpl struct {
	store        store.Store
	certProvider certificate.Provider
	cache        *storebased.Cache[*storage.InitBundleMeta]
}

func (b *backendImpl) GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error) {
	if err := access.CheckAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return nil, err
	}

	allBundleMetas, err := b.store.GetAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving all init bundles")
	}
	return allBundleMetas, nil
}

func extractUserIdentity(ctx context.Context) *storage.User {
	ctxIdentity := authn.IdentityFromContextOrNil(ctx)
	if ctxIdentity == nil {
		return nil
	}

	var providerID string
	var attributes []*storage.UserAttribute

	if provider := ctxIdentity.ExternalAuthProvider(); provider != nil {
		providerID = provider.ID()
	}

	for k, vs := range ctxIdentity.Attributes() {
		for _, v := range vs {
			attributes = append(attributes, &storage.UserAttribute{Key: k, Value: v})
		}
	}

	return &storage.User{
		Id:             ctxIdentity.UID(),
		AuthProviderId: providerID,
		Attributes:     attributes,
	}
}

func extractExpiryDate(certBundle clusters.CertBundle) (*types.Timestamp, error) {
	sensorCert := certBundle[storage.ServiceType_SENSOR_SERVICE]
	if sensorCert == nil {
		return nil, errors.New("no sensor certificate in init bundle")
	}
	timestamp, err := types.TimestampProto(sensorCert.X509Cert.NotAfter)
	if err != nil {
		return nil, errors.Wrap(err, "converting expiry date to timestamp")
	}
	return timestamp, nil
}

func (b *backendImpl) Issue(ctx context.Context, name string) (*InitBundleWithMeta, error) {
	if err := access.CheckAccess(ctx, storage.Access_READ_WRITE_ACCESS); err != nil {
		return nil, err
	}

	if err := validateName(name); err != nil {
		return nil, err
	}

	caCert, err := b.certProvider.GetCA()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving CA certificate")
	}

	user := extractUserIdentity(ctx)
	certBundle, id, err := b.certProvider.GetBundle()
	if err != nil {
		return nil, errors.Wrap(err, "generating certificates for init bundle")
	}

	expiryDate, err := extractExpiryDate(certBundle)
	if err != nil {
		return nil, errors.Wrap(err, "extracting expiry date of certificate bundle")
	}

	meta := &storage.InitBundleMeta{
		Id:        id.String(),
		Name:      name,
		CreatedAt: types.TimestampNow(),
		CreatedBy: user,
		ExpiresAt: expiryDate,
	}

	if err := b.store.Add(ctx, meta); err != nil {
		return nil, errors.Wrap(err, "adding new init bundle to data store")
	}

	return &InitBundleWithMeta{
		CAConfig: CAConfig{
			CACert: caCert,
		},
		CertBundle: certBundle,
		Meta:       meta,
	}, nil
}

func (b *backendImpl) GetCAConfig(ctx context.Context) (*CAConfig, error) {
	if err := access.CheckAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return nil, err
	}

	caCert, err := b.certProvider.GetCA()
	if err != nil {
		return nil, err
	}

	return &CAConfig{
		CACert: caCert,
	}, nil
}

func (b *backendImpl) Revoke(ctx context.Context, id string) error {
	if err := access.CheckAccess(ctx, storage.Access_READ_WRITE_ACCESS); err != nil {
		return err
	}

	if err := b.store.Revoke(ctx, id); err != nil {
		return errors.Wrapf(err, "revoking init bundle %q", id)
	}

	b.cache.InvalidateCache(id)

	return nil
}

func (b *backendImpl) CheckRevoked(ctx context.Context, id string) error {
	if err := access.CheckAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return err
	}

	bundleMeta, err := b.cache.GetObject(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "retrieving init bundle %q", id)
	}

	if bundleMeta.GetIsRevoked() {
		return ErrInitBundleIsRevoked
	}
	return nil
}

// ValidateClientCertificate validates cert chains in identity extractors defined in authn.ValidateCertChain
func (b *backendImpl) ValidateClientCertificate(ctx context.Context, chain []mtls.CertInfo) error {
	if len(chain) == 0 {
		return errors.New("empty cert chain passed")
	}

	leaf := chain[0]
	bundleID := leaf.Subject.Organization
	// check if leaf cert is part of an init bundle
	if len(bundleID) == 0 {
		log.Debugf("Init bundle ID was not found in certificate %q", leaf.Subject.OrganizationalUnit)
		return nil
	}

	subject := mtls.SubjectFromCommonName(leaf.Subject.CommonName)
	if subject.Identifier == centralsensor.EphemeralInitCertClusterID {
		log.Debug("Not checking revocation for operator-issued init cert.")
		return nil
	}

	if err := b.CheckRevoked(sac.WithAllAccess(ctx), bundleID[0]); err != nil {
		if errors.Is(ErrInitBundleIsRevoked, err) {
			log.Errorf("init bundle cert is revoked: %q", bundleID)
			return errors.Wrapf(err, "init bundle verification failed %q", bundleID[0])
		}
		return errors.Wrapf(err, "failed checking init bundle status %q", bundleID[0])
	}

	return nil
}
