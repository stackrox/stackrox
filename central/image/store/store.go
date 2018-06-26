package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const imageBucket = "images"

// Store provides storage functionality for alerts.
type Store interface {
	GetImage(sha string) (*v1.Image, bool, error)
	GetImages() ([]*v1.Image, error)
	CountImages() (int, error)
	AddImage(image *v1.Image) error
	UpdateImage(image *v1.Image) error
	RemoveImage(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, imageBucket)
	return &storeImpl{
		DB: db,
	}
}
