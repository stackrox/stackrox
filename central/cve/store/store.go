package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.CVE, error)
	Count() (int, error)
	Get(id string) (*storage.CVE, bool, error)
	GetBatch(ids []string) ([]*storage.CVE, []int, error)

	Exists(id string) (bool, error)

	Upsert(cve *storage.CVE) error
	UpsertBatch(cves []*storage.CVE) error

	Delete(id string) error
	DeleteBatch(ids []string) error
}
