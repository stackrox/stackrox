package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/sensor/admission-control/manager"
	"google.golang.org/grpc"
)

type service struct {
	mgr manager.Manager
}

// New creates a new admission control API service
func New(mgr manager.Manager) pkgGRPC.APIService {
	return &service{
		mgr: mgr,
	}
}

func (s *service) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

func (s *service) RegisterServiceServer(*grpc.Server) {}

func (s *service) CustomRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		{
			Route:         "/ready",
			Authorizer:    allow.Anonymous(),
			ServerHandler: http.HandlerFunc(s.handleReady),
			Compression:   false,
		},
	}
}

func (s *service) handleReady(w http.ResponseWriter, req *http.Request) {
	if !s.mgr.IsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintln(w, "not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}
