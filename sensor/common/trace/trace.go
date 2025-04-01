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

// Background creates a context based on context.Background with enriched trace values.
func Background() context.Context {
	return withClusterID(context.Background())
}
