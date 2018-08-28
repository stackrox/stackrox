package service

import (
	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/tokenbased/user"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct{}

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
	authStatus, err := tokenAuthStatus(ctx)
	if err == nil {
		return authStatus, nil
	}

	authStatus, err = tlsAuthStatus(ctx)
	if err == nil {
		return authStatus, nil
	}

	return nil, status.Error(codes.Unauthenticated, "not authenticated")
}

func tokenAuthStatus(ctx context.Context) (*v1.AuthStatus, error) {
	identity, err := authn.FromTokenBasedIdentityContext(ctx)
	if err != nil {
		return nil, err
	}
	exp, err := types.TimestampProto(identity.Expiration())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expiration time: %s", err)
	}
	var url string
	if asUserIdentity, ok := identity.(user.Identity); ok {
		url = asUserIdentity.AuthProvider().RefreshURL()
	}
	return &v1.AuthStatus{
		Id:         &v1.AuthStatus_UserId{UserId: identity.ID()},
		Expires:    exp,
		RefreshUrl: url,
	}, nil
}

func tlsAuthStatus(ctx context.Context) (*v1.AuthStatus, error) {
	id, err := authn.FromTLSContext(ctx)
	switch {
	case err == authn.ErrNoContext:
		return nil, status.Error(codes.Unauthenticated, err.Error())
	case err != nil:
		return nil, status.Error(codes.Internal, err.Error())
	}
	exp, err := types.TimestampProto(id.Expiration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expiration time: %s", err)
	}
	return &v1.AuthStatus{
		Id:      &v1.AuthStatus_ServiceId{ServiceId: id.Identity.V1()},
		Expires: exp,
	}, nil
}
