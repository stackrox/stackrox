package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/and"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = func() authz.Authorizer {
		return perrpc.FromMap(map[authz.Authorizer][]string{
			and.And(
				user.With(permissions.Modify(resources.VulnerabilityManagementRequests)),
				user.With(permissions.Modify(resources.VulnerabilityManagementApprovals))): {
				"/v1.NodeCVEService/SuppressCVEs",
				"/v1.NodeCVEService/UnsuppressCVEs",
			},
		})
	}()
)

// serviceImpl provides APIs for CVEs.
type serviceImpl struct {
	v1.UnimplementedNodeCVEServiceServer

	cves datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNodeCVEServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNodeCVEServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// SuppressCVEs suppresses CVEs from policy workflow and API endpoints that include cve in the responses.
func (s *serviceImpl) SuppressCVEs(ctx context.Context, request *v1.SuppressCVERequest) (*v1.Empty, error) {
	if len(request.GetCves()) == 0 {
		return nil, errox.InvalidArgs.CausedBy("no cves provided to snooze")
	}
	createdAt := time.Now()
	suppressDuration, err := protocompat.DurationFromProto(request.GetDuration())
	if err != nil {
		return nil, err
	}
	if err := s.cves.Suppress(ctx, &createdAt, &suppressDuration, request.GetCves()...); err != nil {
		return nil, err
	}
	// Nodes are not part of policy workflow, and we do not reprocess risk on cve snooze. Hence, nothing to do.
	return &v1.Empty{}, nil
}

// UnsuppressCVEs un-suppresses given node CVEs.
func (s *serviceImpl) UnsuppressCVEs(ctx context.Context, request *v1.UnsuppressCVERequest) (*v1.Empty, error) {
	if len(request.GetCves()) == 0 {
		return nil, errox.InvalidArgs.CausedBy("no cves provided to un-snooze")
	}
	if err := s.cves.Unsuppress(ctx, request.GetCves()...); err != nil {
		return nil, err
	}
	// Nodes are not part of policy workflow, and we do not reprocess risk on cve un-snooze. Hence, nothing to do.
	return &v1.Empty{}, nil
}
