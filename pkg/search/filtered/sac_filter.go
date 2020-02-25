package filtered

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/sac"
)

// GraphProvider is an interface that allows us to interact with an RGraph for the duration of a function's execution.
type GraphProvider interface {
	NewGraphView() graph.DiscardableRGraph
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

// WithResourceHelper uses the input ForResourceHelper to do SAC checks on ourput results.
func WithResourceHelper(resourceHelper sac.ForResourceHelper) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.resourceHelper = &resourceHelper
	}
}

// WithGraphProvider uses the input graph provider for mapping ids to their scopes.
func WithGraphProvider(gp GraphProvider) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.graphProvider = gp
	}
}

// WithClusterPath provides the path in the graph to the cluster scope by the bucket hops
func WithClusterPath(steps ...[]byte) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.clusterPath = steps
	}
}

// WithNamespacePath provides the path in the graph to the namespace scope by the bucket hops.
// Must be the leading portion of the cluster path if provided.
func WithNamespacePath(steps ...[]byte) SACFilterOption {
	return func(filter *filterBuilder) {
		filter.namespacePath = steps
	}
}

type filterBuilder struct {
	resourceHelper *sac.ForResourceHelper
	graphProvider  GraphProvider
	namespacePath  [][]byte
	clusterPath    [][]byte
}

func compile(builder *filterBuilder) (Filter, error) {
	if builder.resourceHelper == nil {
		return nil, errors.New("cannot create a SAC filter without a resource type")
	}
	if builder.graphProvider == nil && builder.clusterPath != nil {
		return nil, errors.New("cannot create a cluster or namespace scoped SAC filter without a graph provider")
	}
	if builder.namespacePath != nil && builder.clusterPath == nil {
		return nil, errors.New("cannot create a namespace scope SAC filter without a cluster path")
	}
	if builder.namespacePath != nil && !isPrefixOf(builder.namespacePath, builder.clusterPath) {
		return nil, errors.New("namespace path must be a sub-path to the cluster path")
	}
	if builder.clusterPath == nil && builder.namespacePath == nil {
		return &globalFilterImpl{
			resourceHelper: *builder.resourceHelper,
		}, nil
	} else if builder.namespacePath == nil {
		return &clusterFilterImpl{
			resourceHelper: *builder.resourceHelper,
			graphProvider:  builder.graphProvider,
			clusterPath:    builder.clusterPath,
		}, nil
	}
	return &namespaceFilterImpl{
		resourceHelper: *builder.resourceHelper,
		graphProvider:  builder.graphProvider,
		clusterPath:    builder.clusterPath,
		namespaceIndex: len(builder.namespacePath) - 1,
	}, nil
}

func isPrefixOf(subpath, path [][]byte) bool {
	if len(subpath) >= len(path) {
		return false
	}
	for idx, step := range subpath {
		if !bytes.Equal(step, path[idx]) {
			return false
		}
	}
	return true
}
