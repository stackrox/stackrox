package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
)

type imageStore struct {
	db.ImageStorage
}

func newImageStore(persistent db.ImageStorage) *imageStore {
	return &imageStore{
		ImageStorage: persistent,
	}
}
