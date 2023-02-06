package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// SignatureIntegrationStore provides storage functionality for signature integrations.
//
//go:generate mockgen-wrapper
type SignatureIntegrationStore interface {
	Get(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error)
	Count(ctx context.Context) (int, error)
	Upsert(ctx context.Context, obj *storage.SignatureIntegration) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.SignatureIntegration) error) error
}
