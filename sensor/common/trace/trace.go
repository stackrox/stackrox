package trace

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/metadata"
)

type clusterIDGetter interface {
	GetNoWait() string
}

// ContextWithClusterID enhances a given context with the Cluster ID
func ContextWithClusterID(ctx context.Context, getter clusterIDGetter) context.Context {
	return metadata.NewOutgoingContext(ctx,
		metadata.Pairs(logging.ClusterIDContextValue, getter.GetNoWait()),
	)
}

// Background creates a context based on context.Background with enriched trace values.
func Background(getter clusterIDGetter) context.Context {
	return ContextWithClusterID(context.Background(), getter)
}
