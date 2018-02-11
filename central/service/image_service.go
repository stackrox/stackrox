package service

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewImageService returns the ImageService API.
func NewImageService(storage db.ImageStorage) *ImageService {
	return &ImageService{
		storage: storage,
	}
}

// ImageService is the struct that manages Images API
type ImageService struct {
	storage db.ImageStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *ImageService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImageServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *ImageService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterImageServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *ImageService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(user.Any().Authorized(ctx))
}

// GetImage returns an image with given sha if it exists.
func (s *ImageService) GetImage(ctx context.Context, request *v1.ResourceByID) (*v1.Image, error) {
	image, exists, err := s.storage.GetImage(request.GetId())
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

// GetImages retrieves all images.
func (s *ImageService) GetImages(ctx context.Context, request *v1.GetImagesRequest) (*v1.GetImagesResponse, error) {
	images, err := s.storage.GetImages(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetImagesResponse{Images: images}, nil
}
