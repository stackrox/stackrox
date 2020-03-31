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

type namespaceFilterImpl struct {
	resourceHelper sac.ForResourceHelper
	graphProvider  GraphProvider
	namespaceIndex int
	clusterPath    [][]byte
}

func (f *namespaceFilterImpl) Apply(ctx context.Context, from ...string) ([]string, error) {
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
		namespacesToClusters := f.collectNamespaceScopes(ctx, idGraph, f.clusterPath[1:], prefixedID)
		ok, err := scopeChecker.AnyAllowed(ctx, convertToNamespaceScopes(namespacesToClusters))
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

func (f *namespaceFilterImpl) collectNamespaceScopes(ctx context.Context, idGraph graph.RGraph, path [][]byte, id []byte) map[string]set.StringSet {
	parents := filterByPrefix(path[0], idGraph.GetRefsTo(id))
	ret := make(map[string]set.StringSet)
	// If this is the namespace index, set the namespace values in the map, and collect all clusters it matches for return.
	if len(f.clusterPath)-len(path) == f.namespaceIndex {
		for _, parentID := range parents {
			namespace := string(dbhelper.StripBucket(path[0], parentID))
			ret[namespace] = f.collectClusterScopes(ctx, idGraph, path[1:], parentID)
		}
		return ret
	}
	// Keep ascending to the namespace.
	for _, parentID := range parents {
		ret = combineMaps(ret, f.collectNamespaceScopes(ctx, idGraph, path[1:], parentID))
	}
	return ret
}

func (f *namespaceFilterImpl) collectClusterScopes(ctx context.Context, idGraph graph.RGraph, path [][]byte, id []byte) set.StringSet {
	parents := filterByPrefix(path[0], idGraph.GetRefsTo(id))
	ret := set.NewStringSet()
	// If this is the cluster index, add it to the set, and return.
	if len(path) == 1 {
		for _, parentID := range parents {
			ret.Add(string(dbhelper.StripBucket(path[0], parentID)))
		}
		return ret
	}
	// Keep ascending to the clusters, collecting them all in the set.
	for _, parentID := range parents {
		ret = ret.Union(f.collectClusterScopes(ctx, idGraph, path[1:], parentID))
	}
	return ret
}

func convertToNamespaceScopes(namespaceToClusters map[string]set.StringSet) [][]sac.ScopeKey {
	ret := make([][]sac.ScopeKey, 0, len(namespaceToClusters))
	for namespace, clusters := range namespaceToClusters {
		for _, cluster := range clusters.AsSlice() {
			ret = append(ret, []sac.ScopeKey{sac.ClusterScopeKey(cluster), sac.NamespaceScopeKey(namespace)})
		}
	}
	return ret
}

func combineMaps(m1, m2 map[string]set.StringSet) map[string]set.StringSet {
	for namespace := range m1 {
		if _, existsInM2 := m2[namespace]; existsInM2 {
			m1[namespace] = m1[namespace].Union(m2[namespace])
		}
	}
	for namespace := range m2 {
		if _, existsInM1 := m1[namespace]; !existsInM1 {
			m1[namespace] = m2[namespace]
		}
	}
	return m1
}
