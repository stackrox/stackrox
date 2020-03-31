package filtered

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

type clusterFilterImpl struct {
	resourceHelper sac.ForResourceHelper
	graphProvider  GraphProvider
	clusterPath    [][]byte
}

func (f *clusterFilterImpl) Apply(ctx context.Context, from ...string) ([]string, error) {
	if ok, err := f.resourceHelper.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return from, nil
	}

	scopeChecker := f.resourceHelper.ScopeChecker(ctx, storage.Access_READ_ACCESS)
	idGraph := f.graphProvider.NewGraphView()
	defer idGraph.Discard()

	// DFS
	errorList := errorhelpers.NewErrorList("errors during SAC filtering")
	filtered := make([]string, 0, len(from))
	for _, id := range from {
		prefixedID := dbhelper.GetBucketKey(f.clusterPath[0], []byte(id))
		clusters := f.collectClusterScopes(ctx, idGraph, f.clusterPath[1:], prefixedID)
		ok, err := scopeChecker.AnyAllowed(ctx, convertToClusterScopes(clusters))
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if ok {
			filtered = append(filtered, id)
		}
	}
	return filtered, errorList.ToError()
}

func (f *clusterFilterImpl) collectClusterScopes(ctx context.Context, idGraph graph.RGraph, path [][]byte, id []byte) set.StringSet {
	// recursively pull out the cluster id, ascending through the provided path.
	parents := filterByPrefix(path[0], idGraph.GetRefsTo(id))
	ret := set.NewStringSet()
	if len(path) == 1 {
		for _, parentID := range parents {
			ret.Add(string(dbhelper.StripBucket(path[0], parentID)))
		}
		return ret
	}
	for _, parentID := range parents {
		ret = ret.Union(f.collectClusterScopes(ctx, idGraph, path[1:], parentID))
	}
	return ret
}

func convertToClusterScopes(clusters set.StringSet) [][]sac.ScopeKey {
	ret := make([][]sac.ScopeKey, 0, len(clusters))
	for _, cluster := range clusters.AsSlice() {
		ret = append(ret, []sac.ScopeKey{sac.ClusterScopeKey(cluster)})
	}
	return ret
}
