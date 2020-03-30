package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	ListImage(sha string) (*storage.ListImage, bool, error)

	GetImages() ([]*storage.Image, error)
	CountImages() (int, error)
	GetImage(sha string, withCVESummaries bool) (*storage.Image, bool, error)
	GetImagesBatch(shas []string) ([]*storage.Image, []int, error)

	Exists(id string) (bool, error)

	Upsert(image *storage.Image) error
	Delete(id string) error

	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)
}
