package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
)

type clusterStore struct {
	db.ClusterStorage
}

func newClusterStore(persistent db.ClusterStorage) *clusterStore {
	return &clusterStore{
		ClusterStorage: persistent,
	}
}
