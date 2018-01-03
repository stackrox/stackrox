package inmem

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
)

type notifierStore struct {
	db.NotifierStorage
}

func newNotifierStore(persistent db.NotifierStorage) *notifierStore {
	return &notifierStore{
		NotifierStorage: persistent,
	}
}
