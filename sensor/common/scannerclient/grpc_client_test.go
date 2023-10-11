package scannerclient

import (
	"context"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/scanner/client/mocks"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_v4Client_GetImageAnalysis(t *testing.T) {
	type args struct {
		image *storage.Image
		cfg   *types.Config
	}
	tests := []struct {
		name    string
		args    args
		setMock func(m *mocks.MockScannerClient)
		want    *ImageAnalysis
		wantErr string
	}{
		{
			name: "when get index fails then return error",
			setMock: func(m *mocks.MockScannerClient) {
				m.EXPECT().
					GetOrCreateImageIndex(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&v4.IndexReport{}, status.Error(codes.Unavailable, "index failed"))
			},
			args: args{
				image: &storage.Image{
					Name: &storage.ImageName{
						FullName: "foobar@sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e",
					},
				},
				cfg: &types.Config{},
			},
			wantErr: "index failed",
		},
		{
			name: "when get index succeeds then return index report",
			setMock: func(m *mocks.MockScannerClient) {
				m.EXPECT().
					GetOrCreateImageIndex(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&v4.IndexReport{
						State:    "IndexFinished",
						Contents: &v4.Contents{},
					}, nil)
			},
			args: args{
				image: &storage.Image{
					Name: &storage.ImageName{
						FullName: "foobar@sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e",
					},
				},
				cfg: &types.Config{},
			},
			want: &ImageAnalysis{
				ScanStatus: scannerV1.ScanStatus_SUCCEEDED,
				V4Contents: &v4.Contents{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewMockScannerClient(gomock.NewController(t))
			tt.setMock(m)
			c := &v4Client{client: m}
			got, err := c.GetImageAnalysis(context.Background(), tt.args.image, tt.args.cfg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
