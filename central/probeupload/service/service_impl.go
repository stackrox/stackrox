package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/probeupload/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ProbeUpload)): {
			"/v1.ProbeUploadService/GetExistingProbes",
		},
	})
)

type service struct {
	mgr manager.Manager
}

func newService(mgr manager.Manager) *service {
	return &service{
		mgr: mgr,
	}
}

func (s *service) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterProbeUploadServiceServer(grpcServer, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterProbeUploadServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) GetExistingProbes(ctx context.Context, req *v1.GetExistingProbesRequest) (*v1.GetExistingProbesResponse, error) {
	fileInfos, err := s.mgr.GetExistingProbeFiles(ctx, req.GetFilesToCheck())
	if err != nil {
		return nil, err
	}
	return &v1.GetExistingProbesResponse{
		ExistingFiles: fileInfos,
	}, nil
}
