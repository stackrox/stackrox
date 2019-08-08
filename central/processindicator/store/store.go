package store

import (
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	GetProcessIndicator(id string) (*storage.ProcessIndicator, bool, error)
	GetProcessIndicators() ([]*storage.ProcessIndicator, error)
	GetProcessInfoToArgs() (map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs, error)
	AddProcessIndicator(*storage.ProcessIndicator) (string, error)
	AddProcessIndicators(...*storage.ProcessIndicator) ([]string, error)
	RemoveProcessIndicator(id string) error
	RemoveProcessIndicators(id []string) error
	GetTxnCount() (txNum uint64, err error)
	IncTxnCount() error
}
