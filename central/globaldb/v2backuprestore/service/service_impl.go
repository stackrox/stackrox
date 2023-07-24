package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	dbAuthz "github.com/stackrox/rox/central/globaldb/authz"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.Authenticated(): {
			"/v1.DBService/GetExportCapabilities",
		},
		dbAuthz.DBReadAccessAuthorizer(): {
			"/v1.DBService/GetActiveRestoreProcess",
		},
		dbAuthz.DBWriteAccessAuthorizer(): {
			"/v1.DBService/CancelRestoreProcess",
			"/v1.DBService/InterruptRestoreProcess",
		},
	})
)

type service struct {
	v1.UnimplementedDBServiceServer

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

func (s *service) GetExportCapabilities(_ context.Context, _ *v1.Empty) (*v1.GetDBExportCapabilitiesResponse, error) {
	return &v1.GetDBExportCapabilitiesResponse{
		Formats:            s.mgr.GetExportFormats().ToProtos(),
		SupportedEncodings: s.mgr.GetSupportedFileEncodings(),
	}, nil
}

func (s *service) GetActiveRestoreProcess(_ context.Context, _ *v1.Empty) (*v1.GetActiveDBRestoreProcessResponse, error) {
	process := s.mgr.GetActiveRestoreProcess()
	if process == nil {
		return &v1.GetActiveDBRestoreProcessResponse{}, nil
	}
	return &v1.GetActiveDBRestoreProcessResponse{
		ActiveStatus: process.ProtoStatus(),
	}, nil
}

func (s *service) CancelRestoreProcess(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	process := s.mgr.GetActiveRestoreProcess()
	if err := validateRestoreProcess(process, req.GetId()); err != nil {
		return nil, err
	}

	process.Cancel()
	if !concurrency.WaitInContext(process.Completion(), ctx) {
		return nil, ctx.Err()
	}
	return &v1.Empty{}, nil
}

func (s *service) InterruptRestoreProcess(ctx context.Context, req *v1.InterruptDBRestoreProcessRequest) (*v1.InterruptDBRestoreProcessResponse, error) {
	process := s.mgr.GetActiveRestoreProcess()
	if err := validateRestoreProcess(process, req.GetProcessId()); err != nil {
		return nil, err
	}

	resumeInfo, err := process.Interrupt(ctx, req.GetAttemptId())
	if err != nil {
		return nil, err
	}
	return &v1.InterruptDBRestoreProcessResponse{
		ResumeInfo: resumeInfo,
	}, nil
}

func validateRestoreProcess(process manager.RestoreProcess, id string) error {
	if process == nil {
		return status.Error(codes.FailedPrecondition, "no restore process is currently in progress")
	}
	if process.Metadata().GetId() != id {
		return status.Errorf(codes.FailedPrecondition, "provided ID %q does not match ID %s of currently active restore process", id, process.Metadata().GetId())
	}
	return nil
}
