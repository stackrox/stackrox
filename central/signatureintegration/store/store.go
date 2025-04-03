package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// SignatureIntegrationStore provides storage functionality for signature integrations.
//
//go:generate mockgen-wrapper
type SignatureIntegrationStore interface {
	Get(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Upsert(ctx context.Context, obj *storage.SignatureIntegration) error
	Delete(ctx context.Context, id ...string) error
	Walk(ctx context.Context, fn func(obj *storage.SignatureIntegration) error) error
}
