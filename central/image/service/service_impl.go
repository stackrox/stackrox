package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/image/datastore"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImageServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterImageServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(user.Any().Authorized(ctx))
}

// GetImage returns an image with given sha if it exists.
func (s *serviceImpl) GetImage(ctx context.Context, request *v1.ResourceByID) (*v1.Image, error) {
	image, exists, err := s.datastore.GetImage(request.GetId())
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		log.Error(err)
		return nil, status.Errorf(codes.NotFound, "image with sha '%s' does not exist", request.GetId())
	}

	return image, nil
}

// ListImages retrieves all images in minimal form.
func (s *serviceImpl) ListImages(ctx context.Context, request *v1.RawQuery) (*v1.ListImagesResponse, error) {
	var err error
	var images []*v1.ListImage
	if request.GetQuery() == "" {
		images, err = s.datastore.ListImages()
	} else {
		parser := &search.QueryParser{}
		parsedQuery, err := parser.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		images, err = s.datastore.SearchListImages(parsedQuery)
	}
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &v1.ListImagesResponse{
		Images: images,
	}, nil
}
