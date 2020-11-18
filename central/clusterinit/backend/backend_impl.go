package backend

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit"
	"github.com/stackrox/rox/central/clusterinit/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	nTokenGenerationAttempts = 1000 // before giving up due to token ID collisions.
)

type backendImpl struct {
	tokenStore datastore.DataStore
}

func (b *backendImpl) GetAll(ctx context.Context) ([]*storage.BootstrapTokenWithMeta, error) {
	tokens, err := b.tokenStore.GetAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving all bootstrap tokens")
	}
	return tokens, nil
}

func (b *backendImpl) Get(ctx context.Context, tokenID string) (*storage.BootstrapTokenWithMeta, error) {
	tokenWithMeta, err := b.tokenStore.Get(ctx, tokenID)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving bootstrap token")
	}
	return tokenWithMeta, nil
}

func (b *backendImpl) tryIssueOnce(ctx context.Context, description string) (*storage.BootstrapTokenWithMeta, error) {
	token, err := clusterinit.GenerateBootstrapToken()
	if err != nil {
		return nil, err
	}

	var user *storage.User
	if ctxIdentity := authn.IdentityFromContext(ctx); ctxIdentity != nil {
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

		user = &storage.User{
			Id:             ctxIdentity.UID(),
			AuthProviderId: providerID,
			Attributes:     attributes,
		}
	}

	tokenWithMeta := &storage.BootstrapTokenWithMeta{
		Token:       []byte(token),
		Id:          token.ID(),
		Description: description,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(time.Now()),
		CreatedBy:   user,
		Active:      true,
	}
	err = b.tokenStore.Add(ctx, tokenWithMeta)
	if err != nil {
		return nil, errors.Wrap(err, "adding new bootstrap token to datastore")
	}

	return tokenWithMeta, nil
}

// Issue returns a new bootstrap token.
func (b *backendImpl) Issue(ctx context.Context, description string) (*storage.BootstrapTokenWithMeta, error) {
	var tokenWithMeta *storage.BootstrapTokenWithMeta
	var err error

	for nAttempt := 0; nAttempt < nTokenGenerationAttempts; nAttempt++ {
		tokenWithMeta, err = b.tryIssueOnce(ctx, description)
		if err == nil {
			return tokenWithMeta, nil
		}
		if !errors.Is(err, datastore.ErrTokenIDCollision) {
			return nil, errors.Wrap(err, "issuing token")
		}
	}

	return nil, utils.Should(errors.Errorf("%d consecutive fingerprint collisions when attempting to generate a bootstrap token", nTokenGenerationAttempts))
}

func (b *backendImpl) SetActive(ctx context.Context, tokenID string, active bool) error {
	return b.tokenStore.SetActive(ctx, tokenID, active)
}

// Revoke revokes a token.
func (b *backendImpl) Revoke(ctx context.Context, tokenID string) error {
	err := b.tokenStore.Delete(ctx, tokenID)
	if err != nil {
		return errors.Wrap(err, "revoking token")
	}
	return nil
}
