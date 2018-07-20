package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const (
	orchShaToRegShaBucket = "orchShaToRegShaBucket"
	imageBucket           = "imageBucket"
	listImageBucket       = "images_list"
)

// Store provides storage functionality for alerts.
//go:generate mockery -name=Store
type Store interface {
	ListImage(sha string) (*v1.ListImage, bool, error)
	ListImages() ([]*v1.ListImage, error)

	GetImages() ([]*v1.Image, error)
	CountImages() (int, error)
	GetImage(sha string) (*v1.Image, bool, error)
	GetImagesBatch(shas []string) ([]*v1.Image, error)

	UpsertImage(image *v1.Image) error
	DeleteImage(sha string) error

	GetRegistrySha(orchSha string) (string, bool, error)
	UpsertRegistrySha(orchSha string, regSha string) error
	DeleteRegistrySha(orchSha string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, orchShaToRegShaBucket)
	bolthelper.RegisterBucket(db, imageBucket)
	bolthelper.RegisterBucket(db, listImageBucket)
	return &storeImpl{
		db: db,
	}
}
