package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/mitre/datastore"
	"google.golang.org/grpc"
)

var (
	// While the data served by these endpoints is globally available knowledge,
	// we limit access to authenticated users only to reduce the surface for DoS
	// attacks. Note that `ListMitreAttackVectors()`'s response size is around
	// 1 MB.
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.Authenticated(): {
			"/v1.MitreAttackService/ListMitreAttackVectors",
			"/v1.MitreAttackService/GetMitreAttackVector",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedMitreAttackServiceServer

	store datastore.AttackReadOnlyDataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterMitreAttackServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterMitreAttackServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) ListMitreAttackVectors(_ context.Context, _ *v1.Empty) (*v1.ListMitreAttackVectorsResponse, error) {
	return &v1.ListMitreAttackVectorsResponse{
		MitreAttackVectors: s.store.GetAll(),
	}, nil
}

func (s *serviceImpl) GetMitreAttackVector(_ context.Context, req *v1.ResourceByID) (*v1.GetMitreVectorResponse, error) {
	vector, err := s.store.Get(req.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetMitreVectorResponse{
		MitreAttackVector: vector,
	}, nil
}
