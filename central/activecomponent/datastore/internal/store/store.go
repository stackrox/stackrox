package store

import (
	"github.com/stackrox/stackrox/central/activecomponent/converter"
	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for active component.
//go:generate mockgen-wrapper
type Store interface {
	Exists(id string) (bool, error)
	Get(id string) (*storage.ActiveComponent, bool, error)
	GetBatch(ids []string) ([]*storage.ActiveComponent, []int, error)

	UpsertBatch(activeComponents []*converter.CompleteActiveComponent) error
	DeleteBatch(ids ...string) error
}
