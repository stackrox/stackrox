package singletons

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/pkg/grpc/clusters"
	"google.golang.org/grpc"
)

var (
	once sync.Once

	grpcUnaryInterceptors  []grpc.UnaryServerInterceptor
	grpcStreamInterceptors []grpc.StreamServerInterceptor
)

func initialize() {
	clusterWatcher := clusters.NewClusterWatcher(clusterDataStore.Singleton())

	grpcUnaryInterceptors = []grpc.UnaryServerInterceptor{
		clusterWatcher.UnaryInterceptor(),
	}
	grpcStreamInterceptors = []grpc.StreamServerInterceptor{
		clusterWatcher.StreamInterceptor(),
	}
}

// GrpcUnaryInterceptors provides the unary interceptors to use with gRPC based services.
func GrpcUnaryInterceptors() []grpc.UnaryServerInterceptor {
	once.Do(initialize)
	return grpcUnaryInterceptors
}

// GrpcStreamInterceptors provides the stream interceptors to use with gRPC based services.
func GrpcStreamInterceptors() []grpc.StreamServerInterceptor {
	once.Do(initialize)
	return grpcStreamInterceptors
}
