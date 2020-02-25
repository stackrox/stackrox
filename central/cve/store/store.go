package store

import (
	"github.com/stackrox/rox/central/cve/converter"
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

	Upsert(cves ...*storage.CVE) error
	UpsertClusterCVEs(cveParts ...converter.ClusterCVEParts) error
	Delete(ids ...string) error
}
