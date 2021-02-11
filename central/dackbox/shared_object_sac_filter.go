package dackbox

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

// SharedObjectSACFilterOption represents an option when creating a SAC filter.
type SharedObjectSACFilterOption func(*combinedSAC)

// WithNode sets the resource helper, scope transformation, and existence check for nodes.
func WithNode(nodeResourceHelper sac.ForResourceHelper, nodeScopeTransform filtered.ScopeTransform, nodeExistenceCheck transformation.OneToBool) SharedObjectSACFilterOption {
	return func(filter *combinedSAC) {
		filter.nodeResourceHelper = &nodeResourceHelper
		filter.nodeScopeTransform = nodeScopeTransform
		filter.nodeExistenceCheck = nodeExistenceCheck
	}
}

// WithImage sets the resource helper, scope transformation, and existence check for images.
func WithImage(imageResourceHelper sac.ForResourceHelper, imageScopeTransform filtered.ScopeTransform, imageExistenceCheck transformation.OneToBool) SharedObjectSACFilterOption {
	return func(filter *combinedSAC) {
		filter.imageResourceHelper = &imageResourceHelper
		filter.imageScopeTransform = imageScopeTransform
		filter.imageExistenceCheck = imageExistenceCheck
	}
}

// WithCluster sets the resource helper, scope transformation, and existence check for clusters.
func WithCluster(clusterResourceHelper sac.ForResourceHelper, clusterScopeTransform filtered.ScopeTransform, clusterExistenceCheck transformation.OneToBool) SharedObjectSACFilterOption {
	return func(filter *combinedSAC) {
		filter.clusterResourceHelper = &clusterResourceHelper
		filter.clusterScopeTransform = clusterScopeTransform
		filter.clusterExistenceCheck = clusterExistenceCheck
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

	if cs.imageResourceHelper == nil || cs.imageScopeTransform == nil || cs.imageExistenceCheck == nil {
		return nil, errors.New("cannot create a SAC filter without proper image entities")
	}
	if cs.nodeResourceHelper == nil || cs.nodeScopeTransform == nil || cs.nodeExistenceCheck == nil {
		return nil, errors.New("cannot create a SAC filter without proper node entities")
	}
	if cs.access == storage.Access_NO_ACCESS {
		return nil, errors.New("cannot create a SAC filter without an access level")
	}

	return cs, nil
}

type combinedSAC struct {
	nodeResourceHelper    *sac.ForResourceHelper
	imageResourceHelper   *sac.ForResourceHelper
	clusterResourceHelper *sac.ForResourceHelper

	nodeScopeTransform    filtered.ScopeTransform
	imageScopeTransform   filtered.ScopeTransform
	clusterScopeTransform filtered.ScopeTransform

	nodeExistenceCheck    transformation.OneToBool
	imageExistenceCheck   transformation.OneToBool
	clusterExistenceCheck transformation.OneToBool

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

func (f *combinedSAC) noSACApply(ctx context.Context, from ...string) []string {
	var hasImageRead, hasNodeRead, hasClusterRead bool
	if id := authn.IdentityFromContext(ctx); id != nil {
		hasImageRead = imageAuthorizer(ctx) == nil
		hasNodeRead = nodeAuthorizer(ctx) == nil
		hasClusterRead = f.clusterResourceHelper != nil && clusterAuthorizer(ctx) == nil
	} else {
		hasImageRead = hasGlobalAccessScope(ctx, f.imageResourceHelper)
		hasNodeRead = hasGlobalAccessScope(ctx, f.nodeResourceHelper)
		hasClusterRead = f.clusterResourceHelper != nil && hasGlobalAccessScope(ctx, f.clusterResourceHelper)
	}

	if hasImageRead && hasNodeRead && (f.clusterResourceHelper == nil || hasClusterRead) {
		return from
	}
	if !hasImageRead && !hasNodeRead && (f.clusterResourceHelper == nil || !hasClusterRead) {
		return nil
	}

	filtered := make([]string, 0, len(from))
	for _, id := range from {
		idBytes := []byte(id)
		if hasImageRead && f.imageExistenceCheck(ctx, idBytes) {
			filtered = append(filtered, id)
			continue
		}
		if hasNodeRead && f.nodeExistenceCheck(ctx, idBytes) {
			filtered = append(filtered, id)
			continue
		}
		if hasClusterRead && f.clusterExistenceCheck(ctx, idBytes) {
			filtered = append(filtered, id)
			continue
		}
	}
	return filtered
}

func (f *combinedSAC) Apply(ctx context.Context, from ...string) ([]string, error) {
	if !sac.IsContextSACEnabled(ctx) {
		return f.noSACApply(ctx, from...), nil
	}

	nodeAccess, err := f.nodeResourceHelper.AccessAllowed(ctx, f.access)
	if err != nil {
		return nil, err
	}

	imageAccess, err := f.imageResourceHelper.AccessAllowed(ctx, f.access)
	if err != nil {
		return nil, err
	}

	nodeScopeChecker := f.nodeResourceHelper.ScopeChecker(ctx, f.access)
	imageScopeChecker := f.imageResourceHelper.ScopeChecker(ctx, f.access)

	var clusterAccess bool
	var clusterScopeChecker sac.ScopeChecker
	if f.clusterResourceHelper != nil {
		clusterAccess, err = f.clusterResourceHelper.AccessAllowed(ctx, f.access)
		if err != nil {
			return nil, err
		}
		clusterScopeChecker = f.clusterResourceHelper.ScopeChecker(ctx, f.access)
	}

	errorList := errorhelpers.NewErrorList("errors during SAC filtering")
	filtered := make([]string, 0, len(from))
	for _, id := range from {
		idBytes := []byte(id)

		if imageAccess {
			// If the image exists and we have image access, then allow
			if exists := f.imageExistenceCheck(ctx, idBytes); exists {
				filtered = append(filtered, id)
				continue
			}
		} else if scopes := f.imageScopeTransform(ctx, idBytes); len(scopes) != 0 {
			ok, err := imageScopeChecker.AnyAllowed(ctx, scopes)
			if err != nil {
				errorList.AddError(err)
				continue
			}
			if ok {
				filtered = append(filtered, id)
				continue
			}
		}

		if scopes := f.nodeScopeTransform(ctx, idBytes); len(scopes) != 0 {
			if nodeAccess {
				filtered = append(filtered, id)
				continue
			}
			ok, err := nodeScopeChecker.AnyAllowed(ctx, scopes)
			if err != nil {
				errorList.AddError(err)
				continue
			}
			if ok {
				filtered = append(filtered, id)
				continue
			}
		}

		if f.clusterResourceHelper != nil {
			if scopes := f.clusterScopeTransform(ctx, idBytes); len(scopes) != 0 {
				if clusterAccess {
					filtered = append(filtered, id)
					continue
				}
				ok, err := clusterScopeChecker.AnyAllowed(ctx, scopes)
				if err != nil {
					errorList.AddError(err)
					continue
				}
				if ok {
					filtered = append(filtered, id)
					continue
				}
			}
		}
	}
	return filtered, errorList.ToError()
}
