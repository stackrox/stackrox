package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	orchShaToRegShaBucket = "orchShaToRegShaBucket"
	imageBucket           = "imageBucket"
	listImageBucket       = "images_list"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper Store
type Store interface {
	ListImage(sha string) (*storage.ListImage, bool, error)
	ListImages() ([]*storage.ListImage, error)

	GetImages() ([]*storage.Image, error)
	CountImages() (int, error)
	GetImage(sha string) (*storage.Image, bool, error)
	GetImagesBatch(shas []string) ([]*storage.Image, error)

	UpsertImage(image *storage.Image) error
	DeleteImage(sha string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, imageBucket)
	bolthelper.RegisterBucketOrPanic(db, listImageBucket)
	return &storeImpl{
		db: db,
	}
}
