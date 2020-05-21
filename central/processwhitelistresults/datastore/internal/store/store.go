package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store implements the interface for process whitelist results
type Store interface {
	Delete(id string) error
	Get(id string) (*storage.ProcessWhitelistResults, bool, error)
	Upsert(whitelistresults *storage.ProcessWhitelistResults) error
}
