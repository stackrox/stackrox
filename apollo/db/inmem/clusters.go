package inmem

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
)

type clusterStore struct {
	db.ClusterStorage
}

func newClusterStore(persistent db.ClusterStorage) *clusterStore {
	return &clusterStore{
		ClusterStorage: persistent,
	}
}
