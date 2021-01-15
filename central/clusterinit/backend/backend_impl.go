package backend

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

type backendImpl struct {
	store store.Store
}

func (b *backendImpl) GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error) {
	if err := checkAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return nil, err
	}

	allBundleMetas, err := b.store.GetAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving all init bundles")
	}
	return allBundleMetas, nil
}

func extractUserIdentity(ctx context.Context) *storage.User {
	ctxIdentity := authn.IdentityFromContext(ctx)
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
	if err := checkAccess(ctx, storage.Access_READ_WRITE_ACCESS); err != nil {
		return nil, err
	}

	user := extractUserIdentity(ctx)
	certBundle, id, err := clusters.IssueSecuredClusterInitCertificates()
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
		CertBundle: certBundle,
		Meta:       meta,
	}, nil
}

func (b *backendImpl) Revoke(ctx context.Context, id string) error {
	if err := checkAccess(ctx, storage.Access_READ_WRITE_ACCESS); err != nil {
		return err
	}

	return errors.New("not implemented")
}

func (b *backendImpl) CheckRevoked(ctx context.Context, id string) error {
	if err := checkAccess(ctx, storage.Access_READ_ACCESS); err != nil {
		return err
	}

	bundleMeta, err := b.store.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "retrieving init bundle")
	}

	if bundleMeta.GetIsRevoked() {
		return ErrInitBundleIsRevoked
	}
	return nil
}
