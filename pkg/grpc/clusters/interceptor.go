// Package clusters provides an interceptor that maintains last-contact-time
// for Cluster Sensors based on their API interactions.
package clusters

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
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
	GetCluster(id string) (*v1.Cluster, bool, error)
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
	return cw.watchUnary
}

// StreamInterceptor parses authentication metadata to maintain the time for
// a cluster's sensor has last contacted this API server.
// Naturally, it should be called after authentication metadata is parsed.
func (cw ClusterWatcher) StreamInterceptor() grpc.StreamServerInterceptor {
	return cw.watchStream
}

func (cw ClusterWatcher) watchUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	err := cw.recordCheckin(ctx)
	if err != nil {
		logger.Warnf("Could not record cluster contact: %v", err)
	}
	return handler(ctx, req)
}

func (cw ClusterWatcher) watchStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := cw.recordCheckin(stream.Context())
	if err != nil {
		logger.Warnf("Could not record cluster contact: %v", err)
	}
	return handler(srv, stream)
}

func (cw ClusterWatcher) recordCheckin(ctx context.Context) error {
	id, err := authn.FromTLSContext(ctx)
	switch {
	case err == authn.ErrNoContext:
		return nil
	case err != nil:
		return err
	}

	if id.Subject.ServiceType != v1.ServiceType_SENSOR_SERVICE {
		return nil
	}

	if id.Subject.Identifier == "" {
		return status.Error(codes.Unauthenticated, "Cluster ID not provided")
	}

	_, exists, _ := cw.clusters.GetCluster(id.Subject.Identifier)
	if !exists {
		return status.Error(codes.Unauthenticated, "Cluster does not exist")
	}

	return cw.clusters.UpdateClusterContactTime(id.Subject.Identifier, time.Now())
}
