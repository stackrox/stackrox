package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(): {
			"/v1.DBService/GetExportCapabilities",
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

func (s *service) RegisterServiceServer(srv *grpc.Server) {
	v1.RegisterDBServiceServer(srv, s)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDBServiceHandler(ctx, mux, conn)
}

func (s *service) GetExportCapabilities(ctx context.Context, _ *v1.Empty) (*v1.GetDBExportCapabilitiesResponse, error) {
	return &v1.GetDBExportCapabilitiesResponse{
		Formats:            s.mgr.GetExportFormats().ToProtos(),
		SupportedEncodings: s.mgr.GetSupportedFileEncodings(),
	}, nil
}
