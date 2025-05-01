package backend

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/backend/access"
	"github.com/stackrox/rox/central/clusterinit/backend/certificate"
	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

const (
	currentCrsVersion = 1
)

var _ authn.ValidateCertChain = (*backendImpl)(nil)

type backendImpl struct {
	store        store.Store
	certProvider certificate.Provider
}

func (b *backendImpl) GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error) {
	if err := access.CheckAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return nil, err
	}
	storeCtx := getStoreReadContext(ctx)

	allBundleMetas, err := b.store.GetAll(storeCtx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving all init bundles")
	}
	return allBundleMetas, nil
}

func (b *backendImpl) GetAllCRS(ctx context.Context) ([]*storage.InitBundleMeta, error) {
	if err := access.CheckAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return nil, err
	}
	storeCtx := getStoreReadContext(ctx)

	allBundleMetas, err := b.store.GetAllCRS(storeCtx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving all CRSs")
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

func extractCertExpiryDate(cert *mtls.IssuedCert) (time.Time, error) {
	if cert == nil {
		return time.Time{}, errors.New("provided certificate is empty")
	}
	if cert.X509Cert == nil {
		return time.Time{}, errors.New("issued certificate is missing X509 material")
	}
	return cert.X509Cert.NotAfter, nil
}

func extractExpiryDate(certBundle clusters.CertBundle) (time.Time, error) {
	sensorCert := certBundle[storage.ServiceType_SENSOR_SERVICE]
	if sensorCert == nil {
		return time.Time{}, errors.New("no sensor certificate in init bundle")
	}
	expiryDate, err := extractCertExpiryDate(sensorCert)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to extract expiry date from sensor client certificate")
	}
	return expiryDate, nil
}

func (b *backendImpl) Issue(ctx context.Context, name string) (*InitBundleWithMeta, error) {
	if err := access.CheckAccess(ctx, storage.Access_READ_WRITE_ACCESS); err != nil {
		return nil, err
	}
	storeCtx := getStoreReadWriteContext(ctx)

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
		return nil, errors.Wrap(err, "extracting expiry date of newly generated init bundle")
	}

	expiryTimestamp, err := protocompat.ConvertTimeToTimestampOrError(expiryDate)
	if err != nil {
		return nil, errors.Wrap(err, "converting expiry date to timestamp")
	}

	meta := &storage.InitBundleMeta{
		Id:        id.String(),
		Name:      name,
		CreatedAt: protocompat.TimestampNow(),
		CreatedBy: user,
		ExpiresAt: expiryTimestamp,
	}

	if err := b.store.Add(storeCtx, meta); err != nil {
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

func (b *backendImpl) IssueCRS(ctx context.Context, name string) (*CRSWithMeta, error) {
	if err := access.CheckAccess(ctx, storage.Access_READ_WRITE_ACCESS); err != nil {
		return nil, err
	}
	storeCtx := getStoreReadWriteContext(ctx)

	if err := validateName(name); err != nil {
		return nil, err
	}

	caCert, err := b.certProvider.GetCA()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving CA certificate")
	}

	user := extractUserIdentity(ctx)
	cert, id, err := b.certProvider.GetCRSCert()
	if err != nil {
		return nil, errors.Wrap(err, "generating CRS certificates")
	}

	expiryDate, err := extractCertExpiryDate(cert)
	if err != nil {
		return nil, errors.Wrap(err, "extracting expiry date of CRS certificate")
	}

	expiryTimestamp, err := protocompat.ConvertTimeToTimestampOrError(expiryDate)
	if err != nil {
		return nil, errors.Wrap(err, "converting CRS expiry date to timestamp")
	}

	// On the storage side we are reusing the InitBundleMeta.
	meta := &storage.InitBundleMeta{
		Id:        id.String(),
		Name:      name,
		CreatedAt: protocompat.TimestampNow(),
		CreatedBy: user,
		ExpiresAt: expiryTimestamp,
		Version:   storage.InitBundleMeta_CRS,
	}

	if err := b.store.Add(storeCtx, meta); err != nil {
		return nil, errors.Wrap(err, "adding new CRS metadata to data store")
	}

	return &CRSWithMeta{
		CRS: &crs.CRS{
			CAs:     []string{caCert},
			Cert:    string(cert.CertPEM),
			Key:     string(cert.KeyPEM),
			Version: currentCrsVersion,
		},
		Meta: meta,
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
	storeCtx := getStoreReadWriteContext(ctx)

	if err := b.store.Revoke(storeCtx, id); err != nil {
		return errors.Wrapf(err, "revoking init bundle %q", id)
	}

	return nil
}

func (b *backendImpl) CheckRevoked(ctx context.Context, id string) error {
	if err := access.CheckAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return err
	}
	storeCtx := getStoreReadContext(ctx)

	bundleMeta, err := b.store.Get(storeCtx, id)
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

func getStoreContext(ctx context.Context, accessMode storage.Access) context.Context {
	accessLevels := []storage.Access{accessMode}
	if accessMode == storage.Access_READ_WRITE_ACCESS {
		accessLevels = append(accessLevels, storage.Access_READ_ACCESS)
	}
	return sac.WithGlobalAccessScopeChecker(
		ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(accessLevels...),
			sac.ResourceScopeKeys(resources.InitBundleMeta),
		),
	)
}

func getStoreReadContext(ctx context.Context) context.Context {
	return getStoreContext(ctx, storage.Access_READ_ACCESS)
}

func getStoreReadWriteContext(ctx context.Context) context.Context {
	return getStoreContext(ctx, storage.Access_READ_WRITE_ACCESS)
}
