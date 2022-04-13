package filtered

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/utils"
)

// ScopeTransform defines a transformation that turns a key into all of it's parent scopes.
type ScopeTransform struct {
	Path      dackbox.BucketPath
	ScopeFunc func(context.Context, string) []sac.ScopeKey
	EdgeIndex *int
}

// NewCachedChecker creates a Searcher that performs the SAC checks corresponding to this scope transform.
func (t *ScopeTransform) NewCachedChecker(ctx context.Context, resourceHelper *sac.ForResourceHelper, am storage.Access) dackbox.Searcher {
	sc := resourceHelper.ScopeChecker(ctx, am)
	lastBucket := t.Path.Elements[t.Path.Len()-1]
	pred := func(key []byte) (bool, error) {
		scopeKey := t.ScopeFunc(ctx, lastBucket.GetID(key))
		return sc.Allowed(ctx, scopeKey...)
	}
	searcher := dackbox.NewCachedSearcher(graph.GetGraph(ctx), pred, t.Path)
	if t.EdgeIndex != nil {
		searcher = dackbox.EdgeSearcher(searcher, *t.EdgeIndex)
	}
	return searcher
}

// SACFilterOption represents an option when creating a SAC filter.
type SACFilterOption func(*filterBuilder)

// NewSACFilter generated a new filter with the given SAC options.
func NewSACFilter(opts ...SACFilterOption) (Filter, error) {
	fb := &filterBuilder{}
	for _, opt := range opts {
		opt(fb)
	}
	return compile(fb)
}

// MustCreateNewSACFilter is like NewSACFilter, but panics if the creation fails.
func MustCreateNewSACFilter(opts ...SACFilterOption) Filter {
	filter, err := NewSACFilter(opts...)
	utils.CrashOnError(err)
	return filter
}

// WithResourceHelper uses the input ForResourceHelper to do SAC checks on output results.
func WithResourceHelper(resourceHelper sac.ForResourceHelper) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.resourceHelper = &resourceHelper
	}
}

// WithScopeTransform uses the input scope transform for getting the scopes of keys.
func WithScopeTransform(scopeTransform ScopeTransform) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.scopeTransform = &scopeTransform
	}
}

// WithReadAccess filters out elements the context has no read access to.
func WithReadAccess() SACFilterOption {
	return func(filter *filterBuilder) {
		filter.access = storage.Access_READ_ACCESS
	}
}

type filterBuilder struct {
	resourceHelper *sac.ForResourceHelper
	scopeTransform *ScopeTransform
	access         storage.Access
}

func compile(builder *filterBuilder) (Filter, error) {
	if builder.resourceHelper == nil {
		return nil, errors.New("cannot create a SAC filter without a resource type")
	}
	if builder.access == storage.Access_NO_ACCESS {
		return nil, errors.New("cannot create a SAC filter without a access level")
	}
	if builder.scopeTransform == nil {
		return &globalFilterImpl{
			resourceHelper: *builder.resourceHelper,
			access:         builder.access,
		}, nil
	}
	return &scopedSACFilterImpl{
		resourceHelper: *builder.resourceHelper,
		scopeTransform: *builder.scopeTransform,
		access:         builder.access,
	}, nil
}
