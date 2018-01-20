package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
)

type serviceIdentityStore struct {
	db.ServiceIdentityStorage
}

func newServiceIdentityStore(persistent db.ServiceIdentityStorage) *serviceIdentityStore {
	return &serviceIdentityStore{
		ServiceIdentityStorage: persistent,
	}
}
