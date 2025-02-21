package trace

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/metadata"
)

import "github.com/stackrox/rox/sensor/common/clusterid"

var log = logging.LoggerForModule()

// DefaultContext is a top level context with enriched trace values.
func DefaultContext() context.Context {
	return metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs(logging.ClusterIDContextValue, clusterid.GetNoWait()),
	)
}
