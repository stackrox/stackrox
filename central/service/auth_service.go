package service

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/allow"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewAuthService returns the API for authentication information.
func NewAuthService() *AuthService {
	return &AuthService{}
}

// AuthService is the struct that manages the Auth API
type AuthService struct{}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *AuthService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAuthServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *AuthService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *AuthService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, ReturnErrorCode(allow.Anonymous().Authorized(ctx))
}

// GetAuthStatus retrieves the auth status based on the credentials given to the server.
func (s *AuthService) GetAuthStatus(ctx context.Context, request *empty.Empty) (*v1.AuthStatus, error) {
	authStatus, err := userAuth(ctx)
	if err == nil {
		return authStatus, nil
	}

	authStatus, err = tlsAuth(ctx)
	if err == nil {
		return authStatus, nil
	}

	return nil, status.Error(codes.Unauthenticated, "not authenticated")
}

func userAuth(ctx context.Context) (*v1.AuthStatus, error) {
	id, err := authn.FromUserContext(ctx)
	if err != nil {
		return nil, err
	}
	exp, err := ptypes.TimestampProto(id.Expiration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expiration time: %s", err)
	}
	var url string
	if id.User.AuthProvider != nil {
		url = id.User.AuthProvider.RefreshURL()
	}
	return &v1.AuthStatus{
		Id:         &v1.AuthStatus_UserId{UserId: id.User.ID},
		Expires:    exp,
		RefreshUrl: url,
	}, nil
}

func tlsAuth(ctx context.Context) (*v1.AuthStatus, error) {
	id, err := authn.FromTLSContext(ctx)
	switch {
	case err == authn.ErrNoContext:
		return nil, status.Error(codes.Unauthenticated, err.Error())
	case err != nil:
		return nil, status.Error(codes.Internal, err.Error())
	}
	exp, err := ptypes.TimestampProto(id.Expiration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expiration time: %s", err)
	}
	return &v1.AuthStatus{
		Id:      &v1.AuthStatus_ServiceId{ServiceId: id.Identity.V1()},
		Expires: exp,
	}, nil
}
