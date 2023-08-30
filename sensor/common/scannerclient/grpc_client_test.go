package scannerclient

import (
	"context"
	"testing"

	gogoProto "github.com/gogo/protobuf/types"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type IndexerClientMock struct {
	mock.Mock
}

func (mock *IndexerClientMock) HasIndexReport(ctx context.Context, in *v4.HasIndexReportRequest, opts ...grpc.CallOption) (*gogoProto.Empty, error) {
	args := mock.Called(ctx, in, opts)
	return args.Get(0).(*gogoProto.Empty), args.Error(1)
}

func (mock *IndexerClientMock) CreateIndexReport(ctx context.Context, in *v4.CreateIndexReportRequest, opts ...grpc.CallOption) (*v4.IndexReport, error) {
	args := mock.Called(ctx, in, opts)
	return args.Get(0).(*v4.IndexReport), args.Error(1)
}

func (mock *IndexerClientMock) GetIndexReport(ctx context.Context, in *v4.GetIndexReportRequest, opts ...grpc.CallOption) (*v4.IndexReport, error) {
	args := mock.Called(ctx, in, opts)
	return args.Get(0).(*v4.IndexReport), args.Error(1)
}

func Test_v4Client_GetImageAnalysis(t *testing.T) {
	type args struct {
		image *storage.Image
		cfg   *types.Config
	}
	tests := []struct {
		name    string
		args    args
		setMock func(m *IndexerClientMock)
		want    *ImageAnalysis
		wantErr string
	}{
		{
			name: "when cannot get index",
			setMock: func(m *IndexerClientMock) {
				m.
					On("GetIndexReport", mock.Anything, mock.Anything, mock.Anything).
					Return(&v4.IndexReport{}, status.Error(codes.Unavailable, "index failed"))
			},
			wantErr: "index failed",
		},
		{
			name: "when index does not exist and create fails",
			setMock: func(m *IndexerClientMock) {
				m.
					On("GetIndexReport", mock.Anything, mock.Anything, mock.Anything).
					Return(&v4.IndexReport{}, status.Error(codes.NotFound, "not found"))
				m.
					On("CreateIndexReport", mock.Anything, mock.Anything, mock.Anything).
					Return(&v4.IndexReport{}, status.Error(codes.Internal, "create failed"))
			},
			args: args{
				cfg: &types.Config{},
			},
			wantErr: "create failed",
		},
		{
			name: "when index exists",
			setMock: func(m *IndexerClientMock) {
				m.
					On("GetIndexReport", mock.Anything, mock.Anything, mock.Anything).
					Return(&v4.IndexReport{
						State:    "IndexFinished",
						Contents: &v4.Contents{},
					}, nil)
			},
			want: &ImageAnalysis{
				ScanStatus: scannerV1.ScanStatus_SUCCEEDED,
				V4Contents: &v4.Contents{},
			},
		},
		{
			name: "when index does not exist and create succeeds",
			setMock: func(m *IndexerClientMock) {
				m.
					On("GetIndexReport", mock.Anything, mock.Anything, mock.Anything).
					Return(&v4.IndexReport{}, status.Error(codes.NotFound, "not found"))
				m.
					On("CreateIndexReport", mock.Anything, mock.MatchedBy(func(r *v4.CreateIndexReportRequest) bool {
						return r.GetHashId() == "/v4/containerimage/fake-image-id"
					}), mock.Anything).
					Return(&v4.IndexReport{
						State:    "IndexFinished",
						Contents: &v4.Contents{},
					}, nil)
			},
			args: args{
				image: &storage.Image{
					Id: "fake-image-id",
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
			m := IndexerClientMock{}
			tt.setMock(&m)
			c := &v4Client{
				IndexerClient: &m,
			}
			got, err := c.GetImageAnalysis(context.Background(), tt.args.image, tt.args.cfg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
