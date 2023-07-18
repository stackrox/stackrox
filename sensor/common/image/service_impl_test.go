package image

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	cacheMocks "github.com/stackrox/rox/pkg/expiringcache/mocks"
	"github.com/stackrox/rox/sensor/common"
	imageMocks "github.com/stackrox/rox/sensor/common/image/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestImageService(t *testing.T) {
	suite.Run(t, new(imageServiceSuite))
}

type imageServiceSuite struct {
	suite.Suite
	mockCtrl          *gomock.Controller
	mockCache         *cacheMocks.MockCache
	mockRegistryStore *imageMocks.MockregistryStore
	mockCentral       *imageMocks.MockcentralClient
	mockLocalScan     *imageMocks.MocklocalScan
	service           *serviceImpl
}

var _ suite.SetupTestSuite = (*imageServiceSuite)(nil)

func (s *imageServiceSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockCache = cacheMocks.NewMockCache(s.mockCtrl)
	s.mockRegistryStore = imageMocks.NewMockregistryStore(s.mockCtrl)
	s.mockCentral = imageMocks.NewMockcentralClient(s.mockCtrl)
	s.mockLocalScan = imageMocks.NewMocklocalScan(s.mockCtrl)
}

func (s *imageServiceSuite) createImageService() {
	s.service = &serviceImpl{
		imageCache:    s.mockCache,
		registryStore: s.mockRegistryStore,
		localScan:     s.mockLocalScan,
		centralReady:  concurrency.NewSignal(),
		centralClient: s.mockCentral,
	}
}

func (s *imageServiceSuite) TestGetImage() {
	ctx := context.Background()
	imageID := "imageID"
	imageName := "imageName"
	err := errors.New("some error")
	errCentral := errors.Wrap(err, "scanning image via central")
	errLocalScan := errors.Wrap(err, "scanning image via local scanner")
	cases := map[string]struct {
		request *sensor.GetImageRequest
		notify  common.SensorComponentEvent
		// Unset expectFn will indicate no call to the function
		expectCache       expectFn
		expectRegistry    expectFn
		expectCentralCall expectFn
		expectLocalScan   expectFn
		expectedError     error
		expectedResponse  *sensor.GetImageResponse
	}{
		"Cache hit and central is unreachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      expectCacheHelper(s.mockCache, 1, createScannedImage(imageName, imageID)),
			expectedError:    nil,
			expectedResponse: createImageResponse(imageName, imageID),
		},
		"Cache hit and central is reachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventCentralReachable,
			expectCache:      expectCacheHelper(s.mockCache, 1, createScannedImage(imageName, imageID)),
			expectedError:    nil,
			expectedResponse: createImageResponse(imageName, imageID),
		},
		"Cache miss and central is unreachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      expectCacheHelper(s.mockCache, 1, nil),
			expectedError:    errCentralNoReachable,
			expectedResponse: nil,
		},
		"Cache miss and central is reachable": {
			request:           createImageRequest(imageName, imageID, false),
			notify:            common.SensorComponentEventCentralReachable,
			expectCache:       expectCacheHelper(s.mockCache, 1, nil),
			expectRegistry:    expectRegistryHelper(s.mockRegistryStore, 1, false),
			expectCentralCall: expectCentralCall(s.mockCentral, 1, createScanImageInternalResponse(imageName, imageID), nil),
			expectedError:     nil,
			expectedResponse:  createImageResponse(imageName, imageID),
		},
		"Cache miss, central is reachable and returns error": {
			request:           createImageRequest(imageName, imageID, false),
			notify:            common.SensorComponentEventCentralReachable,
			expectCache:       expectCacheHelper(s.mockCache, 1, nil),
			expectRegistry:    expectRegistryHelper(s.mockRegistryStore, 1, false),
			expectCentralCall: expectCentralCall(s.mockCentral, 1, nil, err),
			expectedError:     errCentral,
			expectedResponse:  nil,
		},
		"Cache miss, local scan, central is unreachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      expectCacheHelper(s.mockCache, 1, nil),
			expectedError:    errCentralNoReachable,
			expectedResponse: nil,
		},
		"Cache miss, local scan, central is reachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventCentralReachable,
			expectCache:      expectCacheHelper(s.mockCache, 1, nil),
			expectRegistry:   expectRegistryHelper(s.mockRegistryStore, 1, true),
			expectLocalScan:  expectLocalScan(s.mockLocalScan, 1, createScannedImage(imageName, imageID), nil),
			expectedError:    nil,
			expectedResponse: createImageResponse(imageName, imageID),
		},
		"Cache miss, local scan returns error, central is reachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventCentralReachable,
			expectCache:      expectCacheHelper(s.mockCache, 1, nil),
			expectRegistry:   expectRegistryHelper(s.mockRegistryStore, 1, true),
			expectLocalScan:  expectLocalScan(s.mockLocalScan, 1, nil, err),
			expectedError:    errLocalScan,
			expectedResponse: nil,
		},
	}
	for testName, c := range cases {
		s.Run(testName, func() {
			s.createImageService()
			s.service.Notify(c.notify)
			c.expectCache.runIfSet()
			c.expectRegistry.runIfSet()
			c.expectCentralCall.runIfSet()
			c.expectLocalScan.runIfSet()
			res, err := s.service.GetImage(ctx, c.request)
			if c.expectedError != nil {
				s.Assert().EqualError(err, c.expectedError.Error())
			} else {
				s.Assert().NoError(err)
			}
			s.Assert().Equal(c.expectedResponse, res)
		})
	}
}

func expectCacheHelper(mockCache *cacheMocks.MockCache, times int, retValue any) expectFn {
	return func() {
		mockCache.EXPECT().Get(gomock.Any()).Times(times).DoAndReturn(func(_ any) any {
			return retValue
		})
	}
}

func expectRegistryHelper(mockRegistryStore *imageMocks.MockregistryStore, times int, retValue bool) expectFn {
	return func() {
		mockRegistryStore.EXPECT().IsLocal(gomock.Any()).Times(times).DoAndReturn(func(_ any) bool {
			return retValue
		})
	}
}

func expectCentralCall(mockCentral *imageMocks.MockcentralClient, times int, retValue *v1.ScanImageInternalResponse, retErr error) expectFn {
	return func() {
		mockCentral.EXPECT().ScanImageInternal(gomock.Any(), gomock.Any()).Times(times).DoAndReturn(func(_, _ any, _ ...any) (*v1.ScanImageInternalResponse, error) {
			return retValue, retErr
		})
	}
}

func expectLocalScan(mockLocalScan *imageMocks.MocklocalScan, times int, retValue *storage.Image, retErr error) expectFn {
	return func() {
		mockLocalScan.EXPECT().EnrichLocalImageInNamespace(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(times).DoAndReturn(func(_, _, _, _, _, _ any) (*storage.Image, error) {
			return retValue, retErr
		})
	}
}

type expectFn func()

func (f expectFn) runIfSet() {
	if f != nil {
		f()
	}
}

func createScannedImage(name, id string) *storage.Image {
	return &storage.Image{
		Id: id,
		Name: &storage.ImageName{
			FullName: name,
		},
		Scan: &storage.ImageScan{},
	}
}

func createImageRequest(name, id string, scanInline bool) *sensor.GetImageRequest {
	return &sensor.GetImageRequest{
		ScanInline: scanInline,
		Image: &storage.ContainerImage{
			Id: id,
			Name: &storage.ImageName{
				FullName: name,
			},
		},
	}
}

func createScanImageInternalResponse(name, id string) *v1.ScanImageInternalResponse {
	return &v1.ScanImageInternalResponse{
		Image: &storage.Image{
			Id: id,
			Name: &storage.ImageName{
				FullName: name,
			},
			Scan: &storage.ImageScan{},
		},
	}
}

func createImageResponse(name, id string) *sensor.GetImageResponse {
	return &sensor.GetImageResponse{
		Image: &storage.Image{
			Id: id,
			Name: &storage.ImageName{
				FullName: name,
			},
			Scan: &storage.ImageScan{},
		},
	}
}
