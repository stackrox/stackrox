package image

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	cacheMocks "github.com/stackrox/rox/pkg/expiringcache/mocks"
	"github.com/stackrox/rox/sensor/common"
	imageMocks "github.com/stackrox/rox/sensor/common/image/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

func (suite *imageServiceSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockCache = cacheMocks.NewMockCache(suite.mockCtrl)
	suite.mockRegistryStore = imageMocks.NewMockregistryStore(suite.mockCtrl)
	suite.mockCentral = imageMocks.NewMockcentralClient(suite.mockCtrl)
	suite.mockLocalScan = imageMocks.NewMocklocalScan(suite.mockCtrl)
	suite.service = &serviceImpl{
		imageCache:    suite.mockCache,
		registryStore: suite.mockRegistryStore,
		localScan:     suite.mockLocalScan,
		centralReady:  concurrency.NewSignal(),
		centralClient: suite.mockCentral,
	}
}

func (suite *imageServiceSuite) TestGetImage() {
	ctx := context.Background()
	imageID := "imageID"
	imageName := "imageName"
	err := errors.New("some error")
	errCentral := errors.Wrap(err, "scanning image via central")
	errLocalScan := errors.Wrap(errors.New("some error"), "scanning image via local scanner")
	cases := map[string]struct {
		request *sensor.GetImageRequest
		notify  common.SensorComponentEvent
		// Unset expectFunctionHelper will indicate no call to the function
		expectCache       *expectFunctionHelper
		expectRegistry    *expectFunctionHelper
		expectCentralCall *expectFunctionHelper
		expectLocalScan   *expectFunctionHelper
		expectedError     error
		expectedResponse  *sensor.GetImageResponse
	}{
		"Cache hit and central is unreachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      expectCacheHelper(suite.mockCache, 1, createScannedImage(imageName, imageID)),
			expectedError:    nil,
			expectedResponse: createImageResponse(imageName, imageID),
		},
		"Cache hit and central is reachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventCentralReachable,
			expectCache:      expectCacheHelper(suite.mockCache, 1, createScannedImage(imageName, imageID)),
			expectedError:    nil,
			expectedResponse: createImageResponse(imageName, imageID),
		},
		"Cache miss and central is unreachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      expectCacheHelper(suite.mockCache, 1, nil),
			expectedError:    errCentralNoReachable,
			expectedResponse: nil,
		},
		"Cache miss and central is reachable": {
			request:           createImageRequest(imageName, imageID, false),
			notify:            common.SensorComponentEventCentralReachable,
			expectCache:       expectCacheHelper(suite.mockCache, 1, nil),
			expectRegistry:    expectRegistryHelper(suite.mockRegistryStore, 1, false),
			expectCentralCall: expectCentralCall(suite.mockCentral, 1, createScanImageInternalResponse(imageName, imageID), nil),
			expectedError:     nil,
			expectedResponse:  createImageResponse(imageName, imageID),
		},
		"Cache miss, central is reachable and returns error": {
			request:           createImageRequest(imageName, imageID, false),
			notify:            common.SensorComponentEventCentralReachable,
			expectCache:       expectCacheHelper(suite.mockCache, 1, nil),
			expectRegistry:    expectRegistryHelper(suite.mockRegistryStore, 1, false),
			expectCentralCall: expectCentralCall(suite.mockCentral, 1, nil, err),
			expectedError:     errCentral,
			expectedResponse:  nil,
		},
		"Cache miss, local scan, central is unreachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      expectCacheHelper(suite.mockCache, 1, nil),
			expectedError:    errCentralNoReachable,
			expectedResponse: nil,
		},
		"Cache miss, local scan, central is reachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventCentralReachable,
			expectCache:      expectCacheHelper(suite.mockCache, 1, nil),
			expectRegistry:   expectRegistryHelper(suite.mockRegistryStore, 1, true),
			expectLocalScan:  expectLocalScan(suite.mockLocalScan, 1, createScannedImage(imageName, imageID), nil),
			expectedError:    nil,
			expectedResponse: createImageResponse(imageName, imageID),
		},
		"Cache miss, local scan returns error, central is reachable": {
			request:          createImageRequest(imageName, imageID, false),
			notify:           common.SensorComponentEventCentralReachable,
			expectCache:      expectCacheHelper(suite.mockCache, 1, nil),
			expectRegistry:   expectRegistryHelper(suite.mockRegistryStore, 1, true),
			expectLocalScan:  expectLocalScan(suite.mockLocalScan, 1, nil, err),
			expectedError:    errLocalScan,
			expectedResponse: nil,
		},
	}
	for testName, c := range cases {
		suite.T().Run(testName, func(t *testing.T) {
			suite.service.centralReady.Reset()
			suite.service.Notify(c.notify)
			c.expectCache.Fn()()
			c.expectRegistry.Fn()()
			c.expectCentralCall.Fn()()
			c.expectLocalScan.Fn()()
			res, err := suite.service.GetImage(ctx, c.request)
			if c.expectedError != nil {
				assert.EqualError(t, err, c.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.expectedResponse, res)
		})
	}
}

func expectCacheHelper(mockCache *cacheMocks.MockCache, times int, retValue interface{}) *expectFunctionHelper {
	var fn func()
	if times == 0 {
		fn = func() {
			mockCache.EXPECT().Get(gomock.Any()).Times(0)
		}
	} else {
		fn = func() {
			mockCache.EXPECT().Get(gomock.Any()).Times(times).DoAndReturn(func(_ interface{}) interface{} {
				return retValue
			})
		}
	}
	return &expectFunctionHelper{
		fn: fn,
	}
}

func expectRegistryHelper(mockRegistryStore *imageMocks.MockregistryStore, times int, retValue bool) *expectFunctionHelper {
	var fn func()
	if times == 0 {
		fn = func() {
			mockRegistryStore.EXPECT().IsLocal(gomock.Any()).Times(0)
		}
	} else {
		fn = func() {
			mockRegistryStore.EXPECT().IsLocal(gomock.Any()).Times(times).DoAndReturn(func(_ interface{}) bool {
				return retValue
			})
		}
	}
	return &expectFunctionHelper{
		fn: fn,
	}
}

func expectCentralCall(mockCentral *imageMocks.MockcentralClient, times int, retValue *v1.ScanImageInternalResponse, retErr error) *expectFunctionHelper {
	var fn func()
	if times == 0 {
		fn = func() {
			mockCentral.EXPECT().ScanImageInternal(gomock.Any(), gomock.Any()).Times(0)
		}
	} else {
		fn = func() {
			mockCentral.EXPECT().ScanImageInternal(gomock.Any(), gomock.Any()).Times(times).DoAndReturn(func(_, _ interface{}, _ ...interface{}) (*v1.ScanImageInternalResponse, error) {
				return retValue, retErr
			})
		}
	}
	return &expectFunctionHelper{
		fn: fn,
	}
}

func expectLocalScan(mockLocalScan *imageMocks.MocklocalScan, times int, retValue *storage.Image, retErr error) *expectFunctionHelper {
	var fn func()
	if times == 0 {
		fn = func() {
			mockLocalScan.EXPECT().EnrichLocalImageInNamespace(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
	} else {
		fn = func() {
			mockLocalScan.EXPECT().EnrichLocalImageInNamespace(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(times).DoAndReturn(func(_, _, _, _, _, _ interface{}) (*storage.Image, error) {
				return retValue, retErr
			})
		}
	}
	return &expectFunctionHelper{
		fn: fn,
	}
}

type expectFunctionHelper struct {
	fn func()
}

func (e *expectFunctionHelper) Fn() func() {
	if e == nil || e.fn == nil {
		return func() {}
	}
	return e.fn
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
