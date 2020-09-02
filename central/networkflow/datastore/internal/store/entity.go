package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// EntityStore stores network graph entities.
//go:generate mockgen-wrapper
type EntityStore interface {
	GetIDs() ([]string, error)
	GetEntity(id string) (*storage.NetworkEntity, bool, error)

	UpsertEntity(entity *storage.NetworkEntity) error
	DeleteEntity(id string) error
	DeleteEntities(ids []string) error

	Walk(fn func(obj *storage.NetworkEntity) error) error
}
