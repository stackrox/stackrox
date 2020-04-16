package filtered

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

// ScopeTransform defines a transformation that turns a key into all of it's parent scopes.
type ScopeTransform func(ctx context.Context, key []byte) [][]sac.ScopeKey

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

// WithResourceHelper uses the input ForResourceHelper to do SAC checks on ourput results.
func WithResourceHelper(resourceHelper sac.ForResourceHelper) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.resourceHelper = &resourceHelper
	}
}

// WithScopeTransform uses the input scope transform for getting the scopes of keys.
func WithScopeTransform(scopeTransform ScopeTransform) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.scopeTransform = scopeTransform
	}
}

// WithReadAccess filters out elements the context has no read access to.
func WithReadAccess() SACFilterOption {
	return func(filter *filterBuilder) {
		filter.access = storage.Access_READ_ACCESS
	}
}

// WithWriteAccess filters out elements the context has no write access to.
func WithWriteAccess() SACFilterOption {
	return func(filter *filterBuilder) {
		filter.access = storage.Access_READ_WRITE_ACCESS
	}
}

type filterBuilder struct {
	resourceHelper *sac.ForResourceHelper
	scopeTransform ScopeTransform
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
		scopeFunc:      builder.scopeTransform,
		access:         builder.access,
	}, nil
}
