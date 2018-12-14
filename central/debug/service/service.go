package service

import (
	"context"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DebugLogs)): {
			"/v1.DebugService/GetLogLevel",
			"/v1.DebugService/SetLogLevel",
		},
	})
)

// Service provides the interface to the gRPC service for debugging
type Service interface {
	grpcPkg.APIService
	v1.DebugServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a Service that implements v1.DebugServiceServer
func New() Service {
	return &serviceImpl{}
}

type serviceImpl struct{}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDebugServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDebugServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetLogLevel returns a v1.LogLevelResponse object.
func (s *serviceImpl) GetLogLevel(ctx context.Context, req *v1.GetLogLevelRequest) (*v1.LogLevelResponse, error) {
	resp := &v1.LogLevelResponse{}

	// If the request is global, then return all modules who have a log level that does not match the global level
	if len(req.GetModules()) == 0 {
		level := logging.GetGlobalLogLevel()
		resp.Level = logging.LabelForLevelOrInvalid(level)
		logging.ForEachLogger(func(l *logging.Logger) {
			moduleLevel := l.LogLevel()
			if moduleLevel != level {
				resp.ModuleLevels = append(resp.ModuleLevels, &v1.ModuleLevel{Module: l.GetModule(), Level: l.GetLogLevel()})
			}
		})
		return resp, nil
	}

	loggers, unknownModules := logging.GetLoggersByModule(req.GetModules())
	if len(unknownModules) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown module(s): %s", strings.Join(unknownModules, ", "))
	}
	for _, l := range loggers {
		resp.ModuleLevels = append(resp.ModuleLevels, &v1.ModuleLevel{Module: l.GetModule(), Level: l.GetLogLevel()})
	}
	return resp, nil
}

// SetLogLevel implements v1.DebugServiceServer, and it sets the log level for StackRox services.
func (s *serviceImpl) SetLogLevel(ctx context.Context, req *v1.LogLevelRequest) (*types.Empty, error) {
	levelStr := req.GetLevel()
	levelInt, ok := logging.LevelForLabel(levelStr)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown log level %s", levelStr)
	}

	// If this is a global request, then set the global level and return
	if len(req.GetModules()) == 0 {
		logging.SetGlobalLogLevel(levelInt)
		return &types.Empty{}, nil
	}

	loggers, unknownModules := logging.GetLoggersByModule(req.GetModules())
	if len(unknownModules) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown module(s): %s", strings.Join(unknownModules, ", "))
	}

	for _, logger := range loggers {
		logger.SetLogLevel(levelInt)
	}
	return &types.Empty{}, nil
}
