package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	ListAlert(id string) (*storage.ListAlert, bool, error)
	GetListAlerts([]string) ([]*storage.ListAlert, []int, error)

	Walk(fn func(*storage.ListAlert) error) error
	GetIDs() ([]string, error)
	Get(id string) (*storage.Alert, bool, error)
	GetMany(ids []string) ([]*storage.Alert, []int, error)
	Upsert(alert *storage.Alert) error
	Delete(id string) error
	DeleteMany(ids []string) error

	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)
}
