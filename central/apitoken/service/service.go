package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/apitoken/cachedstore"
	"bitbucket.org/stack-rox/apollo/central/apitoken/parser"
	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	rolestore "bitbucket.org/stack-rox/apollo/central/role/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

// Service provides the interface to the svc that handles API keys.
type Service interface {
	v1.APITokenServiceServer

	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a ready-to-use instance of Service.
func New(signer signer.Signer, parser parser.Parser, roleStore rolestore.Store, tokenStore cachedstore.CachedStore) Service {
	return &serviceImpl{signer: signer, parser: parser, roleStore: roleStore, tokenStore: tokenStore}
}
