package image

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/scannerclient"
	"github.com/stackrox/rox/sensor/common/imagecacheutils"
	"google.golang.org/grpc"
)

// Service is an interface to receiving image scan results for the Admission Controller.
type Service interface {
	grpcPkg.APIService
	sensor.ImageServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	SetClient(conn grpc.ClientConnInterface)
}

// NewService returns the ImageService API for the Admission Controller to use.
func NewService(imageCache expiringcache.Cache) Service {
	return &serviceImpl{
		imageCache: imageCache,
	}
}

type serviceImpl struct {
	centralClient v1.ImageServiceClient
	imageCache    expiringcache.Cache
}

func (s *serviceImpl) SetClient(conn grpc.ClientConnInterface) {
	s.centralClient = v1.NewImageServiceClient(conn)
}

func (s *serviceImpl) GetImage(ctx context.Context, req *sensor.GetImageRequest) (*sensor.GetImageResponse, error) {
	if id := req.GetImage().GetId(); id != "" {
		img, _ := s.imageCache.Get(imagecacheutils.GetImageCacheKey(req.GetImage())).(*storage.Image)
		if img != nil && (!req.GetScanInline() || img.GetScan() != nil) {
			return &sensor.GetImageResponse{
				Image: img,
			}, nil
		}
	}

	// Ask Central to scan the image.
	scanResp, err := s.centralClient.ScanImageInternal(ctx, &v1.ScanImageInternalRequest{
		Image:      req.GetImage(),
		CachedOnly: !req.GetScanInline(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "scanning image via central")
	}

	img := scanResp.GetImage()

	// ScanImageInternal may return without error even if it was unable to find the image.
	// Check the metadata here: if Central cannot retrieve the metadata, perhaps the
	// image is stored in an internal registry which Scanner can reach.
	if img.GetMetadata() == nil {
		img, err = scannerclient.ScanImage(ctx, s.centralClient, req.GetImage())
		if err != nil {
			return nil, errors.Wrap(err, "scanning image via local scanner")
		}
	}

	return &sensor.GetImageResponse{
		Image: img,
	}, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterImageServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.AdmissionControlOnly().Authorized(ctx, fullMethodName)
}
