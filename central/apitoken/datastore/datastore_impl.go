package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	apiTokenSAC = sac.ForResource(resources.APIToken)
)

type datastoreImpl struct {
	storage store.Store
}

func (b *datastoreImpl) AddToken(ctx context.Context, token *storage.TokenMetadata) error {
	if ok, err := apiTokenSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return b.storage.AddToken(token)
}

func (b *datastoreImpl) GetTokenOrNil(ctx context.Context, id string) (token *storage.TokenMetadata, err error) {
	if ok, err := apiTokenSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b.storage.GetTokenOrNil(id)
}

func (b *datastoreImpl) GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) (tokens []*storage.TokenMetadata, err error) {
	if ok, err := apiTokenSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b.storage.GetTokens(req)
}

func (b *datastoreImpl) RevokeToken(ctx context.Context, id string) (exists bool, err error) {
	if ok, err := apiTokenSAC.WriteAllowed(ctx); err != nil {
		return false, err
	} else if !ok {
		return false, errors.New("permission denied")
	}

	return b.storage.RevokeToken(id)
}
