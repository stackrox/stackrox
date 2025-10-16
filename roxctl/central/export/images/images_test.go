package images

import (
	"bytes"
	"context"
	"testing"
	"time"

	// Embed is used to import the serialized test object file.
	_ "embed"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/protocompat"
	envMocks "github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/flags"
	ioMocks "github.com/stackrox/rox/roxctl/common/io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

//go:embed serialized_test_image.json
var expectedJSONSerializedImage string

func TestExportImages(t *testing.T) {
	fakeService := &fakeImageService{tb: t}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterImageServiceServer(registrar, fakeService)
		},
	)
	require.NoError(t, err)
	defer closeFunc()

	mockCtrl := gomock.NewController(t)

	var buf bytes.Buffer
	ioMock := ioMocks.NewMockIO(mockCtrl)
	ioMock.EXPECT().Out().Times(1).Return(&buf)

	env := envMocks.NewMockEnvironment(mockCtrl)
	env.EXPECT().GRPCConnection().Times(1).Return(conn, nil)
	env.EXPECT().InputOutput().Times(1).Return(ioMock)

	fakeCmd := &cobra.Command{}
	flags.AddTimeoutWithDefault(fakeCmd, 10*time.Second)

	cmd := Command(env)
	err = cmd.RunE(fakeCmd, []string{})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"image":`+expectedJSONSerializedImage+`}`, buf.String())
}

type fakeImageService struct {
	tb testing.TB
}

func (s *fakeImageService) ExportImages(_ *v1.ExportImageRequest, srv v1.ImageService_ExportImagesServer) error {
	testImage := getImageForSerializationTest()
	eir := &v1.ExportImageResponse{}
	eir.SetImage(testImage)
	return srv.Send(eir)
}

func (s *fakeImageService) GetImage(_ context.Context, _ *v1.GetImageRequest) (*storage.Image, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) CountImages(_ context.Context, _ *v1.RawQuery) (*v1.CountImagesResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) ListImages(_ context.Context, _ *v1.RawQuery) (*v1.ListImagesResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) InvalidateScanAndRegistryCaches(_ context.Context, _ *v1.Empty) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) ScanImageInternal(_ context.Context, _ *v1.ScanImageInternalRequest) (*v1.ScanImageInternalResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) ScanImage(_ context.Context, _ *v1.ScanImageRequest) (*storage.Image, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) GetImageVulnerabilitiesInternal(_ context.Context, _ *v1.GetImageVulnerabilitiesInternalRequest) (*v1.ScanImageInternalResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) EnrichLocalImageInternal(_ context.Context, _ *v1.EnrichLocalImageInternalRequest) (*v1.ScanImageInternalResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) UpdateLocalScanStatusInternal(_ context.Context, _ *v1.UpdateLocalScanStatusInternalRequest) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) DeleteImages(_ context.Context, _ *v1.DeleteImagesRequest) (*v1.DeleteImagesResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) WatchImage(_ context.Context, _ *v1.WatchImageRequest) (*v1.WatchImageResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) UnwatchImage(_ context.Context, _ *v1.UnwatchImageRequest) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}

func (s *fakeImageService) GetWatchedImages(_ context.Context, _ *v1.Empty) (*v1.GetWatchedImagesResponse, error) {
	return nil, errox.NotImplemented
}

// getImageForSerializationTest returns a Mock Image for serialization testing purpose.
func getImageForSerializationTest() *storage.Image {
	// This image is generated in `deployment.go`
	image := fixtures.LightweightDeploymentImage()
	var scanDate = time.Date(2020, time.December, 24, 23, 59, 59, 999999999, time.UTC)
	image.GetScan().SetScanTime(protocompat.ConvertTimeToTimestampOrNil(&scanDate))
	return image
}
