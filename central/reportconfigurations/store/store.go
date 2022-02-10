package store

import "github.com/stackrox/rox/generated/storage"

// Store provides access and update functions for report configurations.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Get(id string) (*storage.ReportConfiguration, bool, error)
	GetMany(ids []string) ([]*storage.ReportConfiguration, []int, error)
	Walk(func(reportConfig *storage.ReportConfiguration) error) error

	Upsert(reportConfig *storage.ReportConfiguration) error
	Delete(id string) error
}
