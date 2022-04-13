package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/apitoken/datastore/internal/store"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	apiTokenSAC = sac.ForResource(resources.APIToken)
)

type datastoreImpl struct {
	storage store.Store

	sync.Mutex
}

func (b *datastoreImpl) AddToken(ctx context.Context, token *storage.TokenMetadata) error {
	if ok, err := apiTokenSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	b.Lock()
	defer b.Unlock()

	return b.storage.Upsert(ctx, token)
}

func (b *datastoreImpl) GetTokenOrNil(ctx context.Context, id string) (token *storage.TokenMetadata, err error) {
	if ok, err := apiTokenSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	b.Lock()
	defer b.Unlock()

	token, exists, err := b.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return token, nil
}

func (b *datastoreImpl) GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error) {
	if ok, err := apiTokenSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	b.Lock()
	defer b.Unlock()

	var tokens []*storage.TokenMetadata
	err := b.storage.Walk(ctx, func(token *storage.TokenMetadata) error {
		if req.GetRevokedOneof() != nil && req.GetRevoked() != token.GetRevoked() {
			return nil
		}
		tokens = append(tokens, token)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (b *datastoreImpl) RevokeToken(ctx context.Context, id string) (bool, error) {
	if ok, err := apiTokenSAC.WriteAllowed(ctx); err != nil {
		return false, err
	} else if !ok {
		return false, sac.ErrResourceAccessDenied
	}

	b.Lock()
	defer b.Unlock()

	token, exists, err := b.storage.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	token.Revoked = true

	if err := b.storage.Upsert(ctx, token); err != nil {
		return false, err
	}
	return true, nil
}
