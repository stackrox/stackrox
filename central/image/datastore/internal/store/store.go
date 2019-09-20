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
	GetImage(sha string) (*storage.Image, bool, error)
	GetImagesBatch(shas []string) ([]*storage.Image, error)

	Exists(id string) (bool, error)

	UpsertImage(image *storage.Image) error
	DeleteImage(id string) error

	GetTxnCount() (txNum uint64, err error)
	IncTxnCount() error
}
