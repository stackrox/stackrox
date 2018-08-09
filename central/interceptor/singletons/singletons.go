package singletons

import (
	"sync"

	"github.com/stackrox/rox/central/apitoken/parser"
	authProviderStore "github.com/stackrox/rox/central/authprovider/cachedstore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/user/mapper"
	"github.com/stackrox/rox/pkg/grpc/authn/tokenbased"
	"github.com/stackrox/rox/pkg/grpc/clusters"
	"google.golang.org/grpc"
)

var (
	once sync.Once

	authInterceptor        *tokenbased.AuthInterceptor
	grpcUnaryInterceptors  []grpc.UnaryServerInterceptor
	grpsStreamInterceptors []grpc.StreamServerInterceptor
)

func initialize() {
	clusterWatcher := clusters.NewClusterWatcher(clusterDataStore.Singleton())

	authInterceptor = tokenbased.NewAuthInterceptor(authProviderStore.Singleton(), usermapper.Singleton(), parser.Singleton())
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
func AuthInterceptor() *tokenbased.AuthInterceptor {
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
