package dackbox

import (
	"context"

	"github.com/pkg/errors"
	clusterDackBox "github.com/stackrox/stackrox/central/cluster/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/grpc/authn"
	"github.com/stackrox/stackrox/pkg/grpc/authz/user"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stackrox/stackrox/pkg/utils"
)

// SharedObjectSACFilterOption represents an option when creating a SAC filter.
type SharedObjectSACFilterOption func(*combinedSAC)

// WithNode sets the resource helper, scope transformation, and existence check for nodes.
func WithNode(nodeResourceHelper sac.ForResourceHelper, pathToNode dackbox.BucketPath) SharedObjectSACFilterOption {
	pathToCluster, err := dackbox.ConcatenatePaths(pathToNode, NodeTransformationPaths[v1.SearchCategory_CLUSTERS])
	if err != nil {
		panic(err)
	}

	scopeTransform := clusterScoped(pathToCluster)
	return func(filter *combinedSAC) {
		filter.nodeResourceHelper = &nodeResourceHelper
		filter.nodeScopeTransform = &scopeTransform
		filter.pathToNode = &pathToNode
	}
}

// WithImage sets the resource helper, scope transformation, and existence check for images.
func WithImage(imageResourceHelper sac.ForResourceHelper, pathToImage dackbox.BucketPath) SharedObjectSACFilterOption {
	pathToNamespace, err := dackbox.ConcatenatePaths(pathToImage, ImageTransformationPaths[v1.SearchCategory_NAMESPACES])
	if err != nil {
		panic(err)
	}

	scopeTransform := namespaceScoped(pathToNamespace)
	return func(filter *combinedSAC) {
		filter.imageResourceHelper = &imageResourceHelper
		filter.imageScopeTransform = &scopeTransform
		filter.pathToImage = &pathToImage
	}
}

// WithCluster sets the resource helper, scope transformation, and existence check for clusters.
func WithCluster(clusterResourceHelper sac.ForResourceHelper, pathToCluster dackbox.BucketPath) SharedObjectSACFilterOption {
	// This should be a no-op, but (a) it better aligns with the other `With...` functions, and (b) we ensure
	// that the given path actually ends at the cluster bucket.
	pathToCluster, err := dackbox.ConcatenatePaths(pathToCluster, dackbox.BackwardsBucketPath(clusterDackBox.BucketHandler))
	if err != nil {
		panic(err)
	}

	scopeTransform := clusterScoped(pathToCluster)
	return func(filter *combinedSAC) {
		filter.clusterResourceHelper = &clusterResourceHelper
		filter.clusterScopeTransform = &scopeTransform
		filter.pathToCluster = &pathToCluster
	}
}

// WithSharedObjectAccess filters out elements the context based on the given context.
func WithSharedObjectAccess(access storage.Access) SharedObjectSACFilterOption {
	return func(filter *combinedSAC) {
		filter.access = access
	}
}

// NewSharedObjectSACFilter returns a filter that is can be used on dependent objects like
// components and vulnerabilities. This is required because images can be orphaned objects that will
// not always have a scope and also even in our traditional RBAC system, we want to filter components
// and cves
func NewSharedObjectSACFilter(opts ...SharedObjectSACFilterOption) (filtered.Filter, error) {
	cs := &combinedSAC{}
	for _, opt := range opts {
		opt(cs)
	}

	if cs.imageResourceHelper == nil || cs.imageScopeTransform == nil || cs.pathToImage == nil {
		return nil, errors.New("cannot create a SAC filter without proper image entities")
	}
	if cs.nodeResourceHelper == nil || cs.nodeScopeTransform == nil || cs.pathToNode == nil {
		return nil, errors.New("cannot create a SAC filter without proper node entities")
	}
	if cs.access == storage.Access_NO_ACCESS {
		return nil, errors.New("cannot create a SAC filter without an access level")
	}

	return cs, nil
}

// MustCreateNewSharedObjectSACFilter is like NewSharedObjectSACFilter, but crashes in case an error occurs.
func MustCreateNewSharedObjectSACFilter(opts ...SharedObjectSACFilterOption) filtered.Filter {
	filter, err := NewSharedObjectSACFilter(opts...)
	utils.CrashOnError(err)
	return filter
}

type combinedSAC struct {
	nodeResourceHelper    *sac.ForResourceHelper
	imageResourceHelper   *sac.ForResourceHelper
	clusterResourceHelper *sac.ForResourceHelper

	nodeScopeTransform    *filtered.ScopeTransform
	imageScopeTransform   *filtered.ScopeTransform
	clusterScopeTransform *filtered.ScopeTransform

	pathToNode    *dackbox.BucketPath
	pathToImage   *dackbox.BucketPath
	pathToCluster *dackbox.BucketPath

	access storage.Access
}

func imageAuthorizer(ctx context.Context) error {
	return user.With(permissions.View(resources.Image)).Authorized(ctx, "sac")
}

func nodeAuthorizer(ctx context.Context) error {
	return user.With(permissions.View(resources.Node)).Authorized(ctx, "sac")
}

func clusterAuthorizer(ctx context.Context) error {
	return user.With(permissions.View(resources.Cluster)).Authorized(ctx, "sac")
}

func hasGlobalAccessScope(ctx context.Context, helper *sac.ForResourceHelper) bool {
	ok, _ := helper.ReadAllowed(ctx)
	return ok
}

func (f *combinedSAC) noSACApply(ctx context.Context, from ...string) ([]int, bool) {
	var hasImageRead, hasNodeRead, hasClusterRead bool
	if id := authn.IdentityFromContextOrNil(ctx); id != nil {
		hasImageRead = imageAuthorizer(ctx) == nil
		hasNodeRead = nodeAuthorizer(ctx) == nil
		hasClusterRead = f.clusterResourceHelper != nil && clusterAuthorizer(ctx) == nil
	} else {
		hasImageRead = hasGlobalAccessScope(ctx, f.imageResourceHelper)
		hasNodeRead = hasGlobalAccessScope(ctx, f.nodeResourceHelper)
		hasClusterRead = f.clusterResourceHelper != nil && hasGlobalAccessScope(ctx, f.clusterResourceHelper)
	}

	if hasImageRead && hasNodeRead && (f.clusterResourceHelper == nil || hasClusterRead) {
		return nil, true
	}
	if !hasImageRead && !hasNodeRead && (f.clusterResourceHelper == nil || !hasClusterRead) {
		return nil, false
	}

	var imageExistenceCheck dackbox.Searcher
	if hasImageRead {
		imageExistenceCheck = dackbox.NewCachedBucketReachabilityChecker(graph.GetGraph(ctx), *f.pathToImage)
	}
	var nodeExistenceCheck dackbox.Searcher
	if hasNodeRead {
		nodeExistenceCheck = dackbox.NewCachedBucketReachabilityChecker(graph.GetGraph(ctx), *f.pathToNode)
	}
	var clusterExistenceCheck dackbox.Searcher
	if hasClusterRead {
		clusterExistenceCheck = dackbox.NewCachedBucketReachabilityChecker(graph.GetGraph(ctx), *f.pathToCluster)
	}

	filteredIndices := make([]int, 0, len(from))
	for idx, id := range from {
		if imageExistenceCheck != nil {
			if allowed, _ := imageExistenceCheck.Search(ctx, id); allowed {
				filteredIndices = append(filteredIndices, idx)
				continue
			}
		}
		if nodeExistenceCheck != nil {
			if allowed, _ := nodeExistenceCheck.Search(ctx, id); allowed {
				filteredIndices = append(filteredIndices, idx)
				continue
			}
		}
		if clusterExistenceCheck != nil {
			if allowed, _ := clusterExistenceCheck.Search(ctx, id); allowed {
				filteredIndices = append(filteredIndices, idx)
				continue
			}
		}
	}
	return filteredIndices, false
}

func (f *combinedSAC) Apply(ctx context.Context, from ...string) ([]int, bool, error) {
	// TODO(ROX-9134): consider re-enabling for Unrestricted scope
	if false {
		filteredIndices, all := f.noSACApply(ctx, from...)
		return filteredIndices, all, nil
	}

	var imageChecker dackbox.Searcher
	if imageAccess, err := f.imageResourceHelper.AccessAllowed(ctx, f.access); err != nil {
		return nil, false, err
	} else if imageAccess {
		// Even if the user has global access to images, we need to ensure that this object (CVE or component)
		// is actually referenced by an image.
		imageChecker = dackbox.NewCachedBucketReachabilityChecker(graph.GetGraph(ctx), *f.pathToImage)
	} else {
		imageChecker = f.imageScopeTransform.NewCachedChecker(ctx, f.imageResourceHelper, f.access)
	}

	var nodeChecker dackbox.Searcher
	if nodeAccess, err := f.nodeResourceHelper.AccessAllowed(ctx, f.access); err != nil {
		return nil, false, err
	} else if nodeAccess {
		// Even if the user has global access to node, we need to ensure that this object (CVE or component)
		// is actually referenced by a node.
		nodeChecker = dackbox.NewCachedBucketReachabilityChecker(graph.GetGraph(ctx), *f.pathToNode)
	} else {
		nodeChecker = f.nodeScopeTransform.NewCachedChecker(ctx, f.nodeResourceHelper, f.access)
	}

	var clusterChecker dackbox.Searcher
	if f.clusterResourceHelper != nil {
		if clusterAccess, err := f.clusterResourceHelper.AccessAllowed(ctx, f.access); err != nil {
			return nil, false, err
		} else if clusterAccess {
			clusterChecker = dackbox.NewCachedBucketReachabilityChecker(graph.GetGraph(ctx), *f.pathToCluster)
		} else {
			clusterChecker = f.clusterScopeTransform.NewCachedChecker(ctx, f.clusterResourceHelper, f.access)
		}
	}

	errorList := errorhelpers.NewErrorList("errors during SAC filtering")
	filteredIndices := make([]int, 0, len(from))

	for idx, id := range from {
		if ok, err := imageChecker.Search(ctx, id); err != nil {
			errorList.AddError(err)
			continue
		} else if ok {
			filteredIndices = append(filteredIndices, idx)
			continue
		}

		if ok, err := nodeChecker.Search(ctx, id); err != nil {
			errorList.AddError(err)
			continue
		} else if ok {
			filteredIndices = append(filteredIndices, idx)
			continue
		}

		if clusterChecker == nil {
			continue
		}

		if ok, err := clusterChecker.Search(ctx, id); err != nil {
			errorList.AddError(err)
			continue
		} else if ok {
			filteredIndices = append(filteredIndices, idx)
			continue
		}
	}
	return filteredIndices, false, errorList.ToError()
}
