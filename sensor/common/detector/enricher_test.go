package detector

import (
	"context"
	"net"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/registry"
	mockStore "github.com/stackrox/rox/sensor/common/store/mocks"
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
	conn, closeFunc := createMockImageService(s.T())
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

func createMockImageService(t *testing.T) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	v1.RegisterImageServiceServer(server,
		&mockImageServiceServer{})
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
}

func (m *mockImageServiceServer) ScanImageInternal(_ context.Context, req *v1.ScanImageInternalRequest) (*v1.ScanImageInternalResponse, error) {
	return &v1.ScanImageInternalResponse{
		Image: types.ToImage(req.Image),
	}, nil
}
