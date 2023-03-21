package image

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/registries/docker"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/tlscheck"
	"github.com/stackrox/rox/sensor/common/imagecacheutils"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/scan"
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
func NewService(imageCache expiringcache.Cache, registryStore *registry.Store) Service {
	return &serviceImpl{
		imageCache:    imageCache,
		registryStore: registryStore,
		localScan:     scan.NewLocalScan(registryStore),
	}
}

type serviceImpl struct {
	sensor.UnimplementedImageServiceServer

	centralClient v1.ImageServiceClient
	imageCache    expiringcache.Cache
	registryStore *registry.Store
	localScan     *scan.LocalScan
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

	// Note: The Admission Controller does NOT know if the image is cluster-local,
	// so we determine it here.
	// If Sensor's registry store has an entry for the given image's registry,
	// it is considered cluster-local.
	req.Image.IsClusterLocal = s.registryStore.HasRegistryForImage(req.GetImage().GetName())

	// Ask Central to scan the image if the image is not internal and local scanning is not forced
	if !req.GetImage().GetIsClusterLocal() && !env.ForceLocalImageScanning.BooleanSetting() {
		scanResp, err := s.centralClient.ScanImageInternal(ctx, &v1.ScanImageInternalRequest{
			Image:      req.GetImage(),
			CachedOnly: !req.GetScanInline(),
		})
		if err != nil {
			return nil, errors.Wrap(err, "scanning image via central")
		}
		return &sensor.GetImageResponse{
			Image: scanResp.GetImage(),
		}, nil
	}

	var err error
	var img *storage.Image
	if req.GetImage().GetIsClusterLocal() {
		img, err = s.localScan.EnrichLocalImage(ctx, s.centralClient, req.GetImage())

	} else {
		// ForceLocalImageScanning must be true
		var reg registryTypes.Registry
		reg, err = s.registryStore.GetFirstRegistryForImage(req.GetImage().GetName())

		if err != nil {
			// no registry found, create a new one temporarily that requires no auth
			reg, err = createNoAuthDockerRegistry(ctx, req.GetImage().GetName().Registry)
			if err != nil {
				return nil, errors.Wrap(err, "creating no-auth image integration")
			}
		}

		img, err = s.localScan.EnrichLocalImageFromRegistry(ctx, s.centralClient, req.GetImage(), reg)
	}

	if err != nil {
		return nil, errors.Wrap(err, "scanning image via local scanner")
	}
	return &sensor.GetImageResponse{
		Image: img,
	}, nil
}

// createNoAuthDockerRegistry creates a new registry image integration of type docker that
// has no credentials
//
// TODO: temporary until image integrations from central are used instead
func createNoAuthDockerRegistry(ctx context.Context, registry string) (registryTypes.Registry, error) {
	secure, err := tlscheck.CheckTLS(ctx, registry)
	if err != nil {
		return nil, err
	}

	reg, err := docker.NewDockerRegistry(&storage.ImageIntegration{
		Id:         registry,
		Name:       registry,
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: registry,
				Insecure: !secure,
			},
		},
	})

	return reg, err
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
