package trace

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/metadata"
)

type clusterIDPeeker interface {
	GetNoWait() string
}

// ContextWithClusterID enhances a given context with the Cluster ID
func ContextWithClusterID(ctx context.Context, clusterID clusterIDPeeker) context.Context {
	return metadata.NewOutgoingContext(ctx,
		metadata.Pairs(logging.ClusterIDContextValue, clusterID.GetNoWait()),
	)
}

// Background creates a context based on context.Background with enriched trace values.
func Background(clusterID clusterIDPeeker) context.Context {
	return ContextWithClusterID(context.Background(), clusterID)
}
