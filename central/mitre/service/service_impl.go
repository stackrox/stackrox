package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/mitre/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"google.golang.org/grpc"
)

var (
	// No permission enforcement since the APIs do not leak any information about RHACS resources,
	// and that MITRE ATT&CK is a globally available knowledge base. Unlike CVEs, we do not add any extra insights to MITRE data.
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.MitreAttackService/ListMitreAttackVectors",
			"/v1.MitreAttackService/GetMitreAttackVector",
		},
	})
)

type serviceImpl struct {
	store datastore.MitreAttackReadOnlyDataStore
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

func (s *serviceImpl) ListMitreAttackVectors(ctx context.Context, _ *v1.Empty) (*v1.ListMitreAttackVectorsResponse, error) {
	return &v1.ListMitreAttackVectorsResponse{
		MitreAttackVectors: s.store.GetAll(),
	}, nil
}

func (s *serviceImpl) GetMitreAttackVector(ctx context.Context, req *v1.ResourceByID) (*v1.GetMitreVectorResponse, error) {
	vector, err := s.store.Get(req.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetMitreVectorResponse{
		MitreAttackVector: vector,
	}, nil
}
