package singletons

import (
	"sync"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	authnUser "bitbucket.org/stack-rox/apollo/pkg/grpc/authn/user"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/clusters"
	"google.golang.org/grpc"
)

var (
	once sync.Once

	authInterceptor        *authnUser.AuthInterceptor
	grpcUnaryInterceptors  []grpc.UnaryServerInterceptor
	grpsStreamInterceptors []grpc.StreamServerInterceptor
)

func initialize() {
	clusterWatcher := clusters.NewClusterWatcher(clusterDataStore.Singleton())

	authInterceptor = authnUser.NewAuthInterceptor()
	grpcUnaryInterceptors = []grpc.UnaryServerInterceptor{
		authInterceptor.UnaryInterceptor(),
		clusterWatcher.UnaryInterceptor(),
	}
	grpsStreamInterceptors = []grpc.StreamServerInterceptor{
		authInterceptor.StreamInterceptor(),
		clusterWatcher.StreamInterceptor(),
	}
}

// AuthInterceptor returns the authInterceptor to provide authentication for requests.
func AuthInterceptor() *authnUser.AuthInterceptor {
	once.Do(initialize)
	return authInterceptor
}

// GrpcUnaryInterceptor provides the unary interceptor to use with GRPC based services.
func GrpcUnaryInterceptor() []grpc.UnaryServerInterceptor {
	once.Do(initialize)
	return grpcUnaryInterceptors
}

// GrpsStreamInterceptors provides the stream interceptor to use with GRPC based services.
func GrpsStreamInterceptors() []grpc.StreamServerInterceptor {
	once.Do(initialize)
	return grpsStreamInterceptors
}
