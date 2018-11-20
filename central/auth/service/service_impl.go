package service

import (
	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/auth/userpass"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	pass *userpass.Issuer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAuthServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAuthServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// GetAuthStatus retrieves the auth status based on the credentials given to the server.
func (s *serviceImpl) GetAuthStatus(ctx context.Context, request *v1.Empty) (*v1.AuthStatus, error) {
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	return authStatusForID(id)
}

func authStatusForID(id authn.Identity) (*v1.AuthStatus, error) {
	exp, err := types.TimestampProto(id.Expiry())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expiration time: %s", err)
	}

	result := &v1.AuthStatus{
		Expires: exp,
	}
	if provider := id.ExternalAuthProvider(); provider != nil {
		result.RefreshUrl = provider.Backend().RefreshURL()
	}
	if svc := id.Service(); svc != nil {
		result.Id = &v1.AuthStatus_ServiceId{ServiceId: svc}
	} else {
		result.Id = &v1.AuthStatus_UserId{UserId: id.UID()}
	}
	return result, nil
}

func (s *serviceImpl) Login(ctx context.Context, request *v1.LoginRequest) (*v1.LoginResponse, error) {
	t, err := s.pass.IssueToken(request.GetUsername(), request.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &v1.LoginResponse{
		Token: t,
	}, nil
}
