package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides the interface to the underlying storage
type Store interface {
	Upsert(ctx context.Context, obj *storage.ComplianceOperatorCheckResult) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.ComplianceOperatorCheckResult) error) error
}
