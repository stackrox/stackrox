package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	ListAlert(id string) (*storage.ListAlert, bool, error)
	ListAlerts() ([]*storage.ListAlert, error)
	GetListAlerts([]string) ([]*storage.ListAlert, []int, error)

	WalkAll(fn func(*storage.ListAlert) error) error
	GetAlertIDs() ([]string, error)
	GetAlert(id string) (*storage.Alert, bool, error)
	GetAlerts(ids []string) ([]*storage.Alert, []int, error)
	UpsertAlert(alert *storage.Alert) error
	DeleteAlert(id string) error
	DeleteAlerts(ids ...string) error

	GetTxnCount() (txNum uint64, err error)
	IncTxnCount() error
}
