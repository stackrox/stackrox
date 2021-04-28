package store

import "github.com/stackrox/rox/generated/storage"

// SimpleAccessScopeStore provides storage functionality for simple access scopes.
//go:generate mockgen-wrapper
type SimpleAccessScopeStore interface {
	Count() (int, error)
	Get(id string) (*storage.SimpleAccessScope, bool, error)
	Upsert(obj *storage.SimpleAccessScope) error
	Delete(id string) error
	Walk(fn func(obj *storage.SimpleAccessScope) error) error
}
