package detector

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/registry"
	mockStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type enricherSuite struct {
	suite.Suite
	mockCtrl                *gomock.Controller
	enricher                *enricher
	mockCache               expiringcache.Cache
	mockServiceAccountStore *mockStore.MockServiceAccountStore
	mockRegistryStore       *registry.Store
}

var _ suite.SetupTestSuite = &enricherSuite{}
var _ suite.TearDownTestSuite = &enricherSuite{}

func (s *enricherSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockCache = expiringcache.NewExpiringCache(env.ReprocessInterval.DurationSetting())
	s.mockServiceAccountStore = mockStore.NewMockServiceAccountStore(s.mockCtrl)
	s.mockRegistryStore = registry.NewRegistryStore(nil)
	s.enricher = newEnricher(s.mockCache,
		s.mockServiceAccountStore,
		s.mockRegistryStore, nil)
}

func (s *enricherSuite) TearDownTest() {
	s.T().Cleanup(s.mockCtrl.Finish)
}

func TestEnricherSuite(t *testing.T) {
	suite.Run(t, new(enricherSuite))
}

func createScanImageRequest(containerID int, imageID string, fullName string, notPullable bool) *scanImageRequest {
	return &scanImageRequest{
		containerIdx: containerID,
		containerImage: &storage.ContainerImage{
			Name: &storage.ImageName{
				FullName: fullName,
			},
			Id:          imageID,
			NotPullable: notPullable,
		},
	}
}

func (s *enricherSuite) Test_dataRaceInRunScan() {
	// Three requests with same Ids but different FullNames
	// The first one should trigger the scan
	req := createScanImageRequest(0, "nginx-id", "nginx:latest", false)
	// The second third should get image from the Cache (getImageFromCache) and set forceEnrichImageWithSignatures
	// to true since they names are different. This will force the detection and should trigger the data race.
	req2 := createScanImageRequest(0, "nginx-id", "quay.io/nginx:latest", false)
	// The third should behave similarly to req2. We added a third request just in case the second is able to
	// bypass getImageFromCache and land to GetOrSet. If that happens, it shouldn't trigger the data race because
	// forceEnrichImageWithSignatures is false and newValue != value so we shouldn't trigger a scan.
	req3 := createScanImageRequest(0, "nginx-id", "nginx:1.14.2", false)
	conn, closeFunc := createMockImageService(s.T(), nil)
	s.enricher.imageSvc = v1.NewImageServiceClient(conn)
	defer closeFunc()
	s.mockCache.RemoveAll()
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(3)
	go func() {
		s.enricher.runScan(req)
		waitGroup.Done()
	}()
	// We wait to make sure the first request finishes
	time.Sleep(2 * time.Second)
	go func() {
		s.enricher.runScan(req2)
		waitGroup.Done()
	}()
	go func() {
		s.enricher.runScan(req3)
		waitGroup.Done()
	}()
	waitGroup.Wait()
}

func createMockImageService(t *testing.T, imageServiceServer v1.ImageServiceServer) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	if imageServiceServer == nil {
		v1.RegisterImageServiceServer(server, &mockImageServiceServer{})
	} else {
		v1.RegisterImageServiceServer(server, imageServiceServer)
	}

	go func() {
		utils.IgnoreError(func() error {
			return server.Serve(listener)
		})
	}()
	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	closeFunc := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}
	return conn, closeFunc
}

type mockImageServiceServer struct {
	v1.UnimplementedImageServiceServer
	callCounts     map[string]int
	callCountsLock sync.Mutex
	returnError    bool
}

func (m *mockImageServiceServer) ScanImageInternal(_ context.Context, req *v1.ScanImageInternalRequest) (*v1.ScanImageInternalResponse, error) {
	if m.callCounts != nil {
		m.callCountsLock.Lock()
		defer m.callCountsLock.Unlock()
		m.callCounts[req.GetImage().GetName().GetFullName()]++
	}

	if m.returnError {
		return nil, errors.New("broken")
	}

	return &v1.ScanImageInternalResponse{
		Image: types.ToImage(req.Image),
	}, nil
}

func (s *enricherSuite) TestScanAndSetWithLock() {
	testutils.MustUpdateFeature(s.T(), features.UnqualifiedSearchRegistries, true)
	testutils.MustUpdateFeature(s.T(), features.SensorSingleScanPerImage, true)

	req := createScanImageRequest(0, "nginx-id", "nginx:latest", false)
	req2 := createScanImageRequest(0, "nginx-id", "quay.io/nginx:latest", false)
	req3 := createScanImageRequest(0, "nginx-id", "nginx:1.14.2", false)
	reqs := []*scanImageRequest{req, req2, req3}

	runScans := func(t *testing.T, imageService *mockImageServiceServer) {
		conn, closeFunc := createMockImageService(s.T(), imageService)
		s.enricher.imageSvc = v1.NewImageServiceClient(conn)
		defer closeFunc()
		s.mockCache.RemoveAll()

		waitGroup := runAsyncScans(s.enricher, reqs)
		waitGroup.Wait()

		// Only a single call per image name should have been made.
		assert.Len(t, imageService.callCounts, 3)
		assert.Equal(t, 1, imageService.callCounts[req.containerImage.GetName().GetFullName()])
		assert.Equal(t, 1, imageService.callCounts[req2.containerImage.GetName().GetFullName()])
		assert.Equal(t, 1, imageService.callCounts[req3.containerImage.GetName().GetFullName()])

		// Simulate a cache expiry.
		s.mockCache.RemoveAll()

		waitGroup = runAsyncScans(s.enricher, reqs)
		waitGroup.Wait()

		// Only one more call per image name should have been made.
		assert.Len(t, imageService.callCounts, 3)
		assert.Equal(t, 2, imageService.callCounts[req.containerImage.GetName().GetFullName()])
		assert.Equal(t, 2, imageService.callCounts[req2.containerImage.GetName().GetFullName()])
		assert.Equal(t, 2, imageService.callCounts[req3.containerImage.GetName().GetFullName()])
	}

	s.T().Run("succesfully scans", func(t *testing.T) {
		imageService := &mockImageServiceServer{callCounts: map[string]int{}}
		runScans(t, imageService)
	})

	s.T().Run("error scans", func(t *testing.T) {
		imageService := &mockImageServiceServer{callCounts: map[string]int{}, returnError: true}
		runScans(t, imageService)
	})
}

func runAsyncScans(e *enricher, reqs []*scanImageRequest) *sync.WaitGroup {
	waitGroup := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		for _, req := range reqs {
			waitGroup.Add(1)
			go func(req *scanImageRequest) {
				e.runScan(req)
				waitGroup.Done()
			}(req)
		}
	}

	return waitGroup
}

func (s *enricherSuite) TestUpdateImageNoLock() {
	name1, _, err := imageUtils.GenerateImageNameFromString("nginx:latest")
	require.NoError(s.T(), err)

	name2, _, err := imageUtils.GenerateImageNameFromString("nginx:1.0")
	require.NoError(s.T(), err)

	name3, _, err := imageUtils.GenerateImageNameFromString("nginx:1.14.2")
	require.NoError(s.T(), err)

	s.T().Run("no panics on nils", func(t *testing.T) {
		var cValue *cacheValue
		assert.NotPanics(t, func() { cValue.updateImageNoLock(nil) })

		cValue = new(cacheValue)
		assert.NotPanics(t, func() { cValue.updateImageNoLock(nil) })
	})

	s.T().Run("do not update cache value on nil image", func(t *testing.T) {
		genCacheValue := func() *cacheValue { return &cacheValue{image: &storage.Image{Name: name1}} }
		cValue := genCacheValue()
		cValue.updateImageNoLock(nil)
		assert.Equal(t, genCacheValue(), cValue)
	})

	s.T().Run("keep existing names when name removed", func(t *testing.T) {
		cValue := &cacheValue{image: &storage.Image{
			Name:  name1,
			Names: []*storage.ImageName{name1, name2},
		}}

		updatedImage := &storage.Image{
			Name:  name2,
			Names: []*storage.ImageName{name2},
		}

		cValue.updateImageNoLock(updatedImage)
		assert.Len(t, cValue.image.Names, 2)
		assert.Contains(t, cValue.image.Names, name1)
		assert.Contains(t, cValue.image.Names, name2)
	})

	s.T().Run("append to names when new one added", func(t *testing.T) {
		cValue := &cacheValue{image: &storage.Image{
			Name:  name1,
			Names: []*storage.ImageName{name1},
		}}

		updatedImage := &storage.Image{
			Name:  name2,
			Names: []*storage.ImageName{name1, name2},
		}

		cValue.updateImageNoLock(updatedImage)
		assert.Len(t, cValue.image.Names, 2)
		assert.Contains(t, cValue.image.Names, name1)
		assert.Contains(t, cValue.image.Names, name2)
	})

	s.T().Run("append to names when new one added and one removed", func(t *testing.T) {
		cValue := &cacheValue{image: &storage.Image{
			Name:  name1,
			Names: []*storage.ImageName{name1, name2},
		}}

		updatedImage := &storage.Image{
			Name:  name2,
			Names: []*storage.ImageName{name1, name3},
		}

		cValue.updateImageNoLock(updatedImage)
		assert.Len(t, cValue.image.Names, 3)
		assert.Contains(t, cValue.image.Names, name1)
		assert.Contains(t, cValue.image.Names, name2)
		assert.Contains(t, cValue.image.Names, name3)
	})
}
