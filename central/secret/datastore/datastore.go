package datastore

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to SecretStorage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchSecrets(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawSecrets(ctx context.Context, q *v1.Query) ([]*storage.Secret, error)
	SearchListSecrets(ctx context.Context, q *v1.Query) ([]*storage.ListSecret, error)

	GetSecret(ctx context.Context, id string) (*storage.Secret, bool, error)
	UpsertSecret(ctx context.Context, request *storage.Secret) error
	RemoveSecret(ctx context.Context, id string) error
}
