package singletons

import (
	"sync"

	authProviderStore "bitbucket.org/stack-rox/apollo/central/authprovider/cachedstore"
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

	authInterceptor = authnUser.NewAuthInterceptor(authProviderStore.Singleton())
	grpcUnaryInterceptors = []grpc.UnaryServerInterceptor{
		authInterceptor.UnaryInterceptor(),
		clusterWatcher.UnaryInterceptor(),
	}
	grpsStreamInterceptors = []grpc.StreamServerInterceptor{
		authInterceptor.StreamInterceptor(),
		clusterWatcher.StreamInterceptor(),
	}
}

// AuthInterceptor provides the auth interceptor to use with gRPC based services.
func AuthInterceptor() *authnUser.AuthInterceptor {
	once.Do(initialize)
	return authInterceptor
}

// GrpcUnaryInterceptors provides the unary interceptors to use with gRPC based services.
func GrpcUnaryInterceptors() []grpc.UnaryServerInterceptor {
	once.Do(initialize)
	return grpcUnaryInterceptors
}

// GrpsStreamInterceptors provides the stream interceptors to use with gRPC based services.
func GrpsStreamInterceptors() []grpc.StreamServerInterceptor {
	once.Do(initialize)
	return grpsStreamInterceptors
}
