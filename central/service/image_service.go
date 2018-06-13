package service

import (
	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewImageService returns the ImageService API.
func NewImageService(datastore datastore.ImageDataStore) *ImageService {
	return &ImageService{
		datastore: datastore,
	}
}

// ImageService is the struct that manages Images API
type ImageService struct {
	datastore datastore.ImageDataStore
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

func convertImagesToListImages(images []*v1.Image) []*v1.ListImage {
	listImages := make([]*v1.ListImage, 0, len(images))
	for _, i := range images {
		listImage := &v1.ListImage{
			Sha:     i.GetName().GetSha(),
			Name:    i.GetName().GetFullName(),
			Created: i.GetMetadata().GetCreated(),
		}

		if i.GetScan() != nil {
			listImage.SetComponents = &v1.ListImage_Components{
				Components: int64(len(i.GetScan().GetComponents())),
			}
			var numVulns int64
			for _, c := range i.GetScan().GetComponents() {
				numVulns += int64(len(c.GetVulns()))
			}
			listImage.SetCves = &v1.ListImage_Cves{
				Cves: numVulns,
			}
		}

		listImages = append(listImages, listImage)
	}
	return listImages
}

// ListImages retrieves all images in minimal form.
func (s *ImageService) ListImages(ctx context.Context, request *v1.RawQuery) (*v1.ListImagesResponse, error) {
	resp := new(v1.ListImagesResponse)
	if request.GetQuery() == "" {
		images, err := s.datastore.GetImages()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Images = convertImagesToListImages(images)
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		images, err := s.datastore.SearchRawImages(parsedQuery)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Images = convertImagesToListImages(images)
	}
	return resp, nil
}
