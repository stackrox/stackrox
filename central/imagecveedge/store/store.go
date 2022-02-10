package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for ImageCVEEdges.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)

	GetAll() ([]*storage.ImageCVEEdge, error)
	Get(id string) (*storage.ImageCVEEdge, bool, error)
	GetMany(ids []string) ([]*storage.ImageCVEEdge, []int, error)
	UpdateVulnState(edgeStates map[string]map[string]storage.VulnerabilityState) error
}
