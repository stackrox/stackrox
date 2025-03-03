package trace

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/metadata"
)

import "github.com/stackrox/rox/sensor/common/clusterid"

var log = logging.LoggerForModule()

func withClusterID(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx,
		metadata.Pairs(logging.ClusterIDContextValue, clusterid.GetNoWait()),
	)
}

// ParentContext creates a top level context with enriched trace values.
func ParentContext() context.Context {
	return withClusterID(context.Background())
}
