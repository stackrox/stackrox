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
	"github.com/stackrox/rox/pkg/expiringcache/mocks"
	"github.com/stackrox/rox/sensor/common"
	mocks2 "github.com/stackrox/rox/sensor/common/image/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestImageService(t *testing.T) {
	suite.Run(t, new(imageServiceSuite))
}

type imageServiceSuite struct {
	suite.Suite
	mockCtrl          *gomock.Controller
	mockCache         *mocks.MockCache
	mockRegistryStore *mocks2.MockregistryStore
	mockCentral       *mocks2.MockcentralClient
	mockLocalScan     *mocks2.MocklocalScan
	service           *serviceImpl
}

var _ suite.SetupTestSuite = (*imageServiceSuite)(nil)

func (suite *imageServiceSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockCache = mocks.NewMockCache(suite.mockCtrl)
	suite.mockRegistryStore = mocks2.NewMockregistryStore(suite.mockCtrl)
	suite.mockCentral = mocks2.NewMockcentralClient(suite.mockCtrl)
	suite.mockLocalScan = mocks2.NewMocklocalScan(suite.mockCtrl)
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
	cases := map[string]struct {
		request *sensor.GetImageRequest
		notify  common.SensorComponentEvent
		// Unset expectValue will indicate no call to the function
		expectCache       *expectValue
		expectRegistry    *expectValue
		expectCentralCall *expectValue
		expectLocalScan   *expectValue
		expectedError     error
		expectedResponse  *sensor.GetImageResponse
	}{
		"Cache hit and central is not reachable": {
			request: createImageRequest("", "123"),
			notify:  common.SensorComponentEventOfflineMode,
			expectCache: createExpectValue(1, &storage.Image{
				Scan: &storage.ImageScan{},
			}, nil),
			expectedError: nil,
			expectedResponse: &sensor.GetImageResponse{
				Image: &storage.Image{
					Scan: &storage.ImageScan{},
				},
			},
		},
		"Cache hit and central is reachable": {
			request: createImageRequest("", "123"),
			notify:  common.SensorComponentEventCentralReachable,
			expectCache: createExpectValue(1, &storage.Image{
				Scan: &storage.ImageScan{},
			}, nil),
			expectedError: nil,
			expectedResponse: &sensor.GetImageResponse{
				Image: &storage.Image{
					Scan: &storage.ImageScan{},
				},
			},
		},
		"No cache hit and central is not reachable": {
			request:          createImageRequest("", "123"),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      createExpectValue(1, nil, nil),
			expectedError:    errCentralNoReachable,
			expectedResponse: nil,
		},
		"No cache hit and central is reachable": {
			request:        createImageRequest("", "123"),
			notify:         common.SensorComponentEventCentralReachable,
			expectCache:    createExpectValue(1, nil, nil),
			expectRegistry: createExpectValue(1, false, nil),
			expectCentralCall: createExpectValue(1, &v1.ScanImageInternalResponse{
				Image: &storage.Image{},
			}, nil),
			expectedError: nil,
			expectedResponse: &sensor.GetImageResponse{
				Image: &storage.Image{},
			},
		},
		"No cache hit, central is reachable and returns error": {
			request:           createImageRequest("", "123"),
			notify:            common.SensorComponentEventCentralReachable,
			expectCache:       createExpectValue(1, nil, nil),
			expectRegistry:    createExpectValue(1, false, nil),
			expectCentralCall: createExpectValue(1, nil, errors.New("some error")),
			expectedError:     errors.Wrap(errors.New("some error"), "scanning image via central"),
			expectedResponse:  nil,
		},
		"No cache hit, local scan, central is not reachable": {
			request:          createImageRequest("", "123"),
			notify:           common.SensorComponentEventOfflineMode,
			expectCache:      createExpectValue(1, nil, nil),
			expectedError:    errCentralNoReachable,
			expectedResponse: nil,
		},
		"No cache hit, local scan, central is reachable": {
			request:         createImageRequest("", "123"),
			notify:          common.SensorComponentEventCentralReachable,
			expectCache:     createExpectValue(1, nil, nil),
			expectRegistry:  createExpectValue(1, true, nil),
			expectLocalScan: createExpectValue(1, &storage.Image{}, nil),
			expectedError:   nil,
			expectedResponse: &sensor.GetImageResponse{
				Image: &storage.Image{},
			},
		},
		"No cache hit, local scan returns error, central is reachable": {
			request:          createImageRequest("", "123"),
			notify:           common.SensorComponentEventCentralReachable,
			expectCache:      createExpectValue(1, nil, nil),
			expectRegistry:   createExpectValue(1, true, nil),
			expectLocalScan:  createExpectValue(1, nil, errors.New("some error")),
			expectedError:    errors.Wrap(errors.New("some error"), "scanning image via local scanner"),
			expectedResponse: nil,
		},
	}
	for testName, c := range cases {
		suite.T().Run(testName, func(t *testing.T) {
			suite.service.centralReady.Reset()
			suite.service.Notify(c.notify)
			if c.expectCache != nil {
				suite.mockCache.EXPECT().Get(gomock.Any()).Times(c.expectCache.times).DoAndReturn(func(_ interface{}) interface{} {
					return c.expectCache.retValue
				})
			} else {
				suite.mockCache.EXPECT().Get(gomock.Any()).Times(0)
			}
			if c.expectRegistry != nil {
				suite.mockRegistryStore.EXPECT().IsLocal(gomock.Any()).Times(c.expectRegistry.times).DoAndReturn(func(_ interface{}) bool {
					ret, ok := c.expectRegistry.retValue.(bool)
					assert.Truef(t, ok, "Invalid test case. The 'ret' value should be of type (bool), got %T", c.expectRegistry.retValue)
					return ret
				})
			} else {
				suite.mockRegistryStore.EXPECT().IsLocal(gomock.Any()).Times(0)
			}
			if c.expectCentralCall != nil {
				suite.mockCentral.EXPECT().ScanImageInternal(gomock.Any(), gomock.Any()).Times(c.expectCentralCall.times).DoAndReturn(func(_, _ interface{}, _ ...interface{}) (*v1.ScanImageInternalResponse, error) {
					if c.expectCentralCall.retValue == nil {
						return nil, c.expectCentralCall.err
					}
					ret, ok := c.expectCentralCall.retValue.(*v1.ScanImageInternalResponse)
					assert.Truef(t, ok, "Invalid test case. The 'ret' value should be of type %T, got %T", &v1.ScanImageInternalResponse{}, c.expectCentralCall.retValue)
					return ret, c.expectCentralCall.err
				})
			} else {
				suite.mockCentral.EXPECT().ScanImageInternal(gomock.Any(), gomock.Any()).Times(0)
			}
			if c.expectLocalScan != nil {
				suite.mockLocalScan.EXPECT().EnrichLocalImageInNamespace(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(c.expectLocalScan.times).DoAndReturn(func(_, _, _, _, _, _ interface{}) (*storage.Image, error) {
					if c.expectLocalScan.retValue == nil {
						return nil, c.expectLocalScan.err
					}
					ret, ok := c.expectLocalScan.retValue.(*storage.Image)
					assert.Truef(t, ok, "Invalid test case. The 'ret' value should be of type %T, got %T", &storage.Image{}, c.expectLocalScan.retValue)
					return ret, c.expectLocalScan.err
				})
			} else {
				suite.mockLocalScan.EXPECT().EnrichLocalImageInNamespace(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}
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

type expectValue struct {
	times    int
	retValue interface{}
	err      error
}

func createExpectValue(times int, val interface{}, err error) *expectValue {
	return &expectValue{
		times:    times,
		retValue: val,
		err:      err,
	}
}

func createImageRequest(name, id string) *sensor.GetImageRequest {
	return &sensor.GetImageRequest{
		ScanInline: false,
		Image: &storage.ContainerImage{
			Id: id,
			Name: &storage.ImageName{
				FullName: name,
			},
		},
	}
}
