// Package clusters provides an interceptor that maintains last-contact-time
// for Cluster Sensors based on their API interactions.
package clusters

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	logger = logging.LoggerForModule()
)

// This interface encapsulates the access the ClusterWatcher needs to
// a cluster store.
type clusterStore interface {
	GetCluster(id string) (*storage.Cluster, bool, error)
	UpdateClusterContactTime(id string, t time.Time) error
}

// A ClusterWatcher provides gRPC interceptors that record cluster checkin
// times based on authentication metadata.
type ClusterWatcher struct {
	clusters clusterStore
}

// NewClusterWatcher creates a new ClusterWatcher.
func NewClusterWatcher(clusters clusterStore) *ClusterWatcher {
	return &ClusterWatcher{
		clusters: clusters,
	}
}

// UnaryInterceptor parses authentication metadata to maintain the time for
// a cluster's sensor has last contacted this API server.
// Naturally, it should be called after authentication metadata is parsed.
func (cw ClusterWatcher) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return contextutil.UnaryServerInterceptor(cw.watch)
}

// StreamInterceptor parses authentication metadata to maintain the time for
// a cluster's sensor has last contacted this API server.
// Naturally, it should be called after authentication metadata is parsed.
func (cw ClusterWatcher) StreamInterceptor() grpc.StreamServerInterceptor {
	return contextutil.StreamServerInterceptor(cw.watch)
}

// watch records a checkin for the cluster obtained from the given context. Its signature satisfies the ContextUpdater
// signature.
func (cw ClusterWatcher) watch(ctx context.Context) (context.Context, error) {
	err := cw.recordCheckin(ctx)
	if err != nil {
		logger.Warnf("Could not record cluster contact: %v", err)
	}
	return ctx, nil
}

func (cw ClusterWatcher) recordCheckin(ctx context.Context) error {
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return nil
	}
	svc := id.Service()
	if svc == nil {
		return nil
	}

	if svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return nil
	}

	if svc.GetId() == "" {
		return status.Error(codes.Unauthenticated, "Cluster ID not provided")
	}

	_, exists, _ := cw.clusters.GetCluster(svc.GetId())
	if !exists {
		return status.Error(codes.Unauthenticated, "Cluster does not exist")
	}

	return cw.clusters.UpdateClusterContactTime(svc.GetId(), time.Now())
}
