package store

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for CVEs.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.CVE, error)
	Count() (int, error)
	Get(id string) (*storage.CVE, bool, error)
	GetBatch(ids []string) ([]*storage.CVE, []int, error)

	Exists(id string) (bool, error)

	Upsert(cves ...*storage.CVE) error
	Delete(ids ...string) error
}
