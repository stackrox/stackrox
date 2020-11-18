package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/backend"
	"github.com/stackrox/rox/central/role"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = user.WithRole(role.Admin)
)

type serviceImpl struct {
	backend backend.Backend
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterClusterInitServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterClusterInitServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetBootstrapTokens(ctx context.Context, empty *v1.Empty) (*v1.BootstrapTokensResponse, error) {
	tokenMetas, err := s.backend.GetAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving meta data for all bootstrap tokens")
	}

	tokens := make([]*v1.BootstrapTokensResponse_BootstrapTokenResponse, 0, len(tokenMetas))
	for _, meta := range tokenMetas {
		tokens = append(tokens, &v1.BootstrapTokensResponse_BootstrapTokenResponse{Id: meta.GetId(), Description: meta.GetDescription()})
	}

	return &v1.BootstrapTokensResponse{Tokens: tokens}, nil
}

func (s *serviceImpl) DeleteBootstrapToken(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	err := s.backend.Revoke(ctx, request.Id)
	if err != nil {
		return nil, errors.Wrap(err, "deleting bootstrap token")
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) PatchBootstrapToken(ctx context.Context, request *v1.BootstrapTokenPatchRequest) (*v1.Empty, error) {
	tokenID := request.GetId()
	if tokenID == "" {
		return &v1.Empty{}, errors.New("no token ID found in patch request")
	}
	if request.GetSetActive() != nil {
		err := s.backend.SetActive(ctx, tokenID, request.GetActive())
		if err != nil {
			return nil, errors.Wrap(err, "patching bootstrap token")
		}
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) PostBootstrapToken(ctx context.Context, request *v1.BootstrapTokenGenRequest) (*v1.BootstrapTokenMetaResponse, error) {
	tokenMeta, err := s.backend.Issue(ctx, request.Description)
	if err != nil {
		return nil, errors.Wrap(err, "generating new bootstrap token")
	}
	rsp := v1.BootstrapTokenMetaResponse{Id: tokenMeta.GetId(), Token: tokenMeta.GetToken(), Description: tokenMeta.GetDescription()}
	return &rsp, nil
}
