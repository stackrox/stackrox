package health

import (
	"context"
	"fmt"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// Provider is any object that can provide health checking information.
type Provider interface {
	// Ready is true when the service can handle traffic.
	Ready() bool
	// Live is true when the service is not stuck.
	Live() bool
	// Name returns the health provider name.
	Name() string
}

// Service is scanner's gRPC API for Health service. It dynamically checks named
// health providers by matching them against service names in the health check
// calls.
type Service struct {
	grpc_health_v1.UnimplementedHealthServer
	providers map[string]Provider
}

// NewService creates a new health gRPC service.
func NewService(list []Provider) *Service {
	providers := make(map[string]Provider, len(list))
	for _, p := range list {
		providers[p.Name()] = p
	}
	return &Service{
		providers: providers,
	}
}

// Check implements grpc_health_v1.HealthCheckRequest.Check
func (s *Service) Check(_ context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	healthy, err := s.check(req.GetService())
	if err != nil {
		return nil, err
	}
	st := grpc_health_v1.HealthCheckResponse_NOT_SERVING
	if healthy {
		st = grpc_health_v1.HealthCheckResponse_SERVING
	}
	return &grpc_health_v1.HealthCheckResponse{Status: st}, nil
}

func (s *Service) check(srv string) (bool, error) {
	var checks []func() bool
	name, probe, _ := strings.Cut(srv, "-")
	if p, ok := s.providers[name]; ok {
		switch probe {
		case "readiness":
			checks = append(checks, p.Ready)
		case "liveness":
			checks = append(checks, p.Live)
		case "":
			checks = append(checks, p.Live, p.Ready)
		default:
			return false, status.Errorf(codes.NotFound, fmt.Sprintf("unknown service: %s", srv))
		}
	} else if name == "" {
		// If the caller does not specify a service name, the server should respond with
		// its overall health status.
		for _, p := range s.providers {
			checks = append(checks, p.Live, p.Ready)
		}
	} else {
		return false, status.Errorf(codes.NotFound, fmt.Sprintf("unknown service: %s", srv))
	}
	// Call all health checks, healthy if all true.
	st := true
	for _, c := range checks {
		st = c() && st
	}
	return st, nil
}

// Watch implements grpc_health_v1.HealthCheckRequest.Watch, which is not supported in Scanner.
func (s *Service) Watch(_ *grpc_health_v1.HealthCheckRequest, _ grpc_health_v1.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *Service) RegisterServiceServer(grpcServer *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *Service) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for this endpoint.
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *Service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// Auth is disabled for health checking endpoints.
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}
