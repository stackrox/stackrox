package store

import "github.com/stackrox/rox/generated/storage"

// SignatureIntegrationStore provides storage functionality for signature integrations.
//go:generate mockgen-wrapper
type SignatureIntegrationStore interface {
	Get(id string) (*storage.SignatureIntegration, bool, error)
	Upsert(obj *storage.SignatureIntegration) error
	Delete(id string) error
	Walk(fn func(obj *storage.SignatureIntegration) error) error
}
