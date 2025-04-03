package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides access and update functions for report configurations.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Get(ctx context.Context, id string) (*storage.ReportConfiguration, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ReportConfiguration, []int, error)
	Walk(context.Context, func(reportConfig *storage.ReportConfiguration) error) error

	Upsert(ctx context.Context, reportConfig *storage.ReportConfiguration) error
	Delete(ctx context.Context, id ...string) error
}
