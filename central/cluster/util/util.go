package util

import (
	"context"
	"errors"

	cluster "github.com/stackrox/rox/central/cluster/datastore"
)

var (
	// ErrClusterNotFound indicates a cluster was not found.
	ErrClusterNotFound = errors.New("cluster not found")
)

// GetClusterIDFromNameOrID returns the ID of cluster. If cluster is already an ID than it is returned
// as is, if cluster is a name will return the associated cluster's ID.
func GetClusterIDFromNameOrID(ctx context.Context, ds cluster.DataStore, cluster string) (string, error) {
	// Clusters will be filtered based on what the user (from ctx) has access to.
	clusters, err := ds.GetClusters(ctx)
	if err != nil {
		return "", err
	}

	// Attempt to match by ID first, do not attempt to match by name yet in case a cluster's name matches
	// the ID of another cluster (unlikely but possible).
	for _, c := range clusters {
		if c.GetId() == cluster {
			return c.GetId(), nil
		}
	}

	// Attempt to match by name.
	for _, c := range clusters {
		// Cluster names are case sensitive such that `REMOTE`, `remote`, and `REMotE` could be different clusters,
		// as a result match the name exactly as provided.
		if c.GetName() == cluster {
			return c.GetId(), nil
		}
	}

	return "", ErrClusterNotFound
}
