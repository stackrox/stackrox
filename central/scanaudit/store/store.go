package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type Store interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Upsert(ctx context.Context, event *storage.ScanAudit) error
	GetMany(ctx context.Context, identifiers []string) ([]*storage.ScanAudit, []int, error)
	GetAll(ctx context.Context) ([]*storage.ScanAudit, error)
}
