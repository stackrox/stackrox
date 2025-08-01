package image

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registrymirror"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/image/cache"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	"google.golang.org/grpc"
)

var (
	log                   = logging.LoggerForModule()
	errCentralNoReachable = errors.New("central is not reachable")
)

// Service is an interface to receiving image scan results for the Admission Controller.
type Service interface {
	grpcPkg.APIService
	sensor.ImageServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	SetClient(conn grpc.ClientConnInterface)
}

// ServiceComponent aggregates the Image Service with the common.SensorComponent
type ServiceComponent interface {
	Service
	common.SensorComponent
}

// NewService returns the ImageService API for the Admission Controller to use.
func NewService(imageCache cache.Image, registryStore registryStore, mirrorStore registrymirror.Store) ServiceComponent {
	return &serviceImpl{
		imageCache:    imageCache,
		registryStore: registryStore,
		localScan:     scan.NewLocalScan(registryStore, mirrorStore),
		centralReady:  concurrency.NewSignal(),
	}
}

type serviceImpl struct {
	unimplemented.Receiver

	sensor.UnimplementedImageServiceServer

	centralClient centralClient
	imageCache    cache.Image
	registryStore registryStore
	localScan     localScan
	centralReady  concurrency.Signal
}

func (s *serviceImpl) Name() string {
	return "image.serviceImpl"
}

func (s *serviceImpl) SetClient(conn grpc.ClientConnInterface) {
	s.centralClient = v1.NewImageServiceClient(conn)
}

func (s *serviceImpl) GetImage(ctx context.Context, req *sensor.GetImageRequest) (*sensor.GetImageResponse, error) {
	id := req.GetImage().GetId()
	v, ok := s.imageCache.Get(cache.GetKey(req.GetImage()))
	if id != "" && ok {
		img := v.GetIfDone()
		if img != nil && (!req.GetScanInline() || img.GetScan() != nil) {
			return &sensor.GetImageResponse{
				Image: img,
			}, nil
		}
	}

	log.Debugf("scan request from admission control: %+v", req)

	// Beyond this point we need to be able to reach central
	if !s.centralReady.IsDone() {
		return nil, errCentralNoReachable
	}

	// Note: The Admission Controller does NOT know if the image is cluster-local,
	// so we determine it here.
	// If Sensor's registry store has an entry for the given image's registry,
	// it is considered cluster-local.
	// This is used to determine that an image is from an OCP internal registry and
	// should not be sent to central for scanning
	req.Image.IsClusterLocal = s.registryStore.IsLocal(req.GetImage().GetName())

	// Ask Central to scan the image if the image is neither internal nor local
	if !req.GetImage().GetIsClusterLocal() {
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

	img, err := s.localScan.EnrichLocalImageInNamespace(ctx, s.centralClient, &scan.LocalScanRequest{
		Image:     req.GetImage(),
		Namespace: req.GetNamespace(),
	})
	if err != nil {
		err = errors.Wrap(err, "scanning image via local scanner")
		log.Error(err)
		return nil, err
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
	err := idcheck.AdmissionControlOnly().Authorized(ctx, fullMethodName)
	return ctx, errors.Wrapf(err, "image authorization for %q", fullMethodName)
}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		s.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		s.centralReady.Reset()
	}
}

func (s *serviceImpl) Start() error {
	return nil
}

func (s *serviceImpl) Stop() {}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (s *serviceImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}
