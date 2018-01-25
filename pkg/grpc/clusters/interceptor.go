// Package clusters provides an interceptor that maintains last-contact-time
// for Cluster Sensors based on their API interactions.
package clusters

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/auth"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"google.golang.org/grpc"
)

var (
	logger = logging.New("grpc/clusters")
)

// ContactTimeUpdater contains the storage-access functions that this
// interceptor requires.
type ContactTimeUpdater interface {
	UpdateClusterContactTime(id string, t time.Time) error
}

// A ClusterWatcher provides gRPC interceptors that record cluster checkin
// times based on authentication metadata.
type ClusterWatcher struct {
	db ContactTimeUpdater
}

// NewClusterWatcher creates a new ClusterWatcher.
func NewClusterWatcher(storage ContactTimeUpdater) *ClusterWatcher {
	return &ClusterWatcher{
		db: storage,
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
	id, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}
	if id.IdentityType.ServiceType == v1.ServiceType_SENSOR_SERVICE && id.Identifier != "" {
		return cw.db.UpdateClusterContactTime(id.Identifier, time.Now())
	}
	return nil
}
