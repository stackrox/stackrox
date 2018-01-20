package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
)

type notifierStore struct {
	db.NotifierStorage
}

func newNotifierStore(persistent db.NotifierStorage) *notifierStore {
	return &notifierStore{
		NotifierStorage: persistent,
	}
}
