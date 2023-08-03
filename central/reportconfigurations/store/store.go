package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides access and update functions for report configurations.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.ReportConfiguration, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ReportConfiguration, []int, error)
	Walk(context.Context, func(reportConfig *storage.ReportConfiguration) error) error
	Exists(ctx context.Context, id string) (bool, error)

	Upsert(ctx context.Context, reportConfig *storage.ReportConfiguration) error
	Delete(ctx context.Context, id string) error
}
