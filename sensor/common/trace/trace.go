package trace

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"google.golang.org/grpc/metadata"
)

func withClusterID(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx,
		metadata.Pairs(logging.ClusterIDContextValue, clusterid.GetNoWait()),
	)
}

// Context creates a top level context with enriched trace values.
func Context() context.Context {
	return withClusterID(context.Background())
}
