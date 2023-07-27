package datastore

import (
	"context"

	"github.com/pkg/errors"
	cluStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
)

var (
	clusterReader = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Cluster))
)

func getClusterIDs(ctx context.Context) (set.StringSet, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, clusterReader)

	clusters, err := cluStore.Singleton().GetClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cluster datastore failure")
	}
	ids := set.NewStringSet()
	for _, cluster := range clusters {
		ids.Add(cluster.GetId())
	}
	return ids, nil
}
