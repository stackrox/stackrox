package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Risk)): {
			"/v1.RiskService/GetRisk",
		},
	})
)

type serviceImpl struct {
	riskDataStore datastore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRiskServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterRiskServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetRisk(ctx context.Context, request *v1.GetRiskRequest) (*storage.Risk, error) {
	subjectType, err := datastore.SubjectType(request.SubjectType)
	if err != nil || subjectType == storage.RiskSubjectType_UNKNOWN {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	risk, err := s.riskDataStore.GetRisk(ctx, request.GetSubjectID(), subjectType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if risk == nil {
		return nil, status.Errorf(codes.NotFound, "risk for %s %s does not exist", request.GetSubjectType(), request.GetSubjectID())
	}
	return risk, nil
}
