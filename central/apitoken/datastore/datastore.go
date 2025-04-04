package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/apitoken/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
)

// DataStore is the gateway to the DB that enforces access control.
type DataStore interface {
	GetTokenOrNil(ctx context.Context, id string) (token *storage.TokenMetadata, err error)
	GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error)

	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	SearchRawTokens(ctx context.Context, q *v1.Query) ([]*storage.TokenMetadata, error)

	AddToken(ctx context.Context, token *storage.TokenMetadata) error
	RevokeToken(ctx context.Context, id string) (exists bool, err error)

	GetNotificationSchedule(ctx context.Context) (*storage.NotificationSchedule, bool, error)
	UpsertNotificationSchedule(ctx context.Context, schedule *storage.NotificationSchedule) error
}

// New returns a ready-to-use DataStore instance.
func New(storage store.Store) DataStore {
	return &datastoreImpl{storage: storage}
}

// NewPostgres returns a ready-to-use DataStore instance plugged to postgres.
func NewPostgres(pool postgres.DB) DataStore {
	return newPostgres(pool)
}

// NewTestPostgres provides a datastore connected to postgres for testing purposes.
func NewTestPostgres(_ testing.TB, pool postgres.DB) DataStore {
	return New(pgStore.New(pool))
}
