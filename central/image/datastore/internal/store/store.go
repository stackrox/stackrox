package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	imageBucket     = []byte("imageBucket")
	listImageBucket = []byte("images_list")
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
	DeleteImage(id string) error

	GetTxnCount() (txNum uint64, err error)
	IncTxnCount() error
}

// New returns a new Store instance using the provided bolt DB instance.
// noUpdateTimestamps controls whether timestamps are automatically updated
// whenever an image is upserted.
func New(db *bolt.DB, noUpdateTimestamps bool) Store {
	bolthelper.RegisterBucketOrPanic(db, imageBucket)
	bolthelper.RegisterBucketOrPanic(db, listImageBucket)
	wrapper, err := bolthelper.NewBoltWrapper(db, imageBucket)
	if err != nil {
		panic(err)
	}
	return &storeImpl{
		db:                 wrapper,
		noUpdateTimestamps: noUpdateTimestamps,
	}
}
