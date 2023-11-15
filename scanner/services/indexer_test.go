package services

import (
	"context"
	"errors"
	"testing"

	"github.com/quay/claircore"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/scanner/indexer/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	imageURL = "https://foobar:443/image:latest"
	hashID   = "/v4/containerimage/foobar"
)

type indexerServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	indexerMock *mocks.MockIndexer
	service     *indexerService
	mockCtrl    *gomock.Controller
}

func TestIndexerServiceSuite(t *testing.T) {
	suite.Run(t, new(indexerServiceTestSuite))
}

func (s *indexerServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.indexerMock = mocks.NewMockIndexer(s.mockCtrl)
	s.service = NewIndexerService(s.indexerMock)
}

func (s *indexerServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func createRequest(id, url, username string) *v4.CreateIndexReportRequest {
	return &v4.CreateIndexReportRequest{
		HashId: id,
		ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
			ContainerImage: &v4.ContainerImageLocator{
				Url:      url,
				Username: username,
			},
		},
	}
}

func (s *indexerServiceTestSuite) setupMock(hashID string, optCount int, report *claircore.IndexReport, err error) {
	s.indexerMock.
		EXPECT().
		IndexContainerImage(gomock.Any(), gomock.Eq(hashID), gomock.Eq(imageURL), gomock.Len(optCount)).
		Return(report, err)
}

func (s *indexerServiceTestSuite) Test_CreateIndexReport_whenUsername_thenAuthEnabled() {
	s.setupMock(hashID, 1, &claircore.IndexReport{}, nil)
	req := createRequest(hashID, imageURL, "sample username")
	r, err := s.service.CreateIndexReport(s.ctx, req)
	s.NoError(err)
	s.Equal(&v4.IndexReport{HashId: hashID, Contents: &v4.Contents{}}, r)
}

func (s *indexerServiceTestSuite) Test_CreateIndexReport_whenNoUsername_thenAuthDisabled() {
	s.setupMock(hashID, 0, &claircore.IndexReport{}, nil)
	req := createRequest(hashID, imageURL, "")
	r, err := s.service.CreateIndexReport(s.ctx, req)
	s.NoError(err)
	s.Equal(&v4.IndexReport{HashId: hashID, Contents: &v4.Contents{}}, r)
}

func (s *indexerServiceTestSuite) Test_CreateIndexReport_whenIndexerError_thenInternalError() {
	s.setupMock(hashID, 0, nil, errors.New(`indexer said "ouch"`))
	req := createRequest(hashID, imageURL, "")
	r, err := s.service.CreateIndexReport(s.ctx, req)
	s.ErrorContains(err, "ouch")
	s.Nil(r)
}

func (s *indexerServiceTestSuite) Test_CreateIndexReport_whenDigest_thenNoError() {
	//#nosec G101 -- This is a false positive
	iURL := "https://foobar:443/image:sha256@sha256:3d44fa76c2c83ed9296e4508b436ff583397cac0f4bad85c2b4ecc193ddb5106"
	s.indexerMock.
		EXPECT().
		IndexContainerImage(gomock.Any(), gomock.Any(), gomock.Eq(iURL), gomock.Len(0)).
		Return(&claircore.IndexReport{}, nil)
	req := createRequest(hashID, iURL, "")
	r, err := s.service.CreateIndexReport(s.ctx, req)
	s.NoError(err)
	s.Equal(&v4.IndexReport{HashId: hashID, Contents: &v4.Contents{}}, r)
}

func (s *indexerServiceTestSuite) Test_CreateIndexReport_InvalidInput() {
	type args struct {
		req *v4.CreateIndexReportRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *v4.IndexReport
		wantErr string
	}{
		{
			name:    "when empty request",
			wantErr: `empty request`,
		},
		{
			name: "when unknown resource type",
			args: args{req: &v4.CreateIndexReportRequest{
				HashId:          "foobar",
				ResourceLocator: nil,
			}},
			wantErr: `invalid hash id: "foobar"`,
		},
		{
			name: "when empty request",
			args: args{req: &v4.CreateIndexReportRequest{
				HashId:          "foobar",
				ResourceLocator: nil,
			}},
			wantErr: `invalid hash id: "foobar"`,
		},
		{
			name: "when empty resource locator",
			args: args{
				req: &v4.CreateIndexReportRequest{
					HashId:          "/v4/containerimage/foobar",
					ResourceLocator: nil,
				},
			},
			wantErr: "invalid resource locator",
		},
		{
			name: "when empty container image URL",
			args: args{
				req: &v4.CreateIndexReportRequest{
					HashId: "/v4/containerimage/foobar",
					ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
						ContainerImage: &v4.ContainerImageLocator{
							Url:      "",
							Username: "",
							Password: "",
						},
					},
				},
			},
			wantErr: "missing image URL",
		},
		{
			name: "when invalid container image URL",
			args: args{
				req: &v4.CreateIndexReportRequest{
					HashId: "/v4/containerimage/foobar",
					ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
						ContainerImage: &v4.ContainerImageLocator{
							Url: "sample-url",
						},
					},
				},
			},
			wantErr: "image URL does not start with",
		},
		{
			name: "when invalid image reference in container image URL",
			args: args{
				req: &v4.CreateIndexReportRequest{
					HashId: "/v4/containerimage/foobar",
					ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
						ContainerImage: &v4.ContainerImageLocator{
							Url: "https://invalid-image-reference",
						},
					},
				},
			},
			wantErr: "could not parse reference: invalid-image-reference",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.service.CreateIndexReport(s.ctx, tt.args.req)
			s.Equal(tt.want, got)
			if tt.wantErr == "" {
				s.NoError(err)
			} else {
				s.ErrorContains(err, tt.wantErr)
			}
		})
	}
}

func (s *indexerServiceTestSuite) Test_GetIndexReport() {
	req := &v4.GetIndexReportRequest{HashId: hashID}

	// When get index report returns an error.
	s.indexerMock.
		EXPECT().
		GetIndexReport(gomock.Any(), gomock.Eq(hashID)).
		Return(nil, false, errors.New("ouch"))
	r, err := s.service.GetIndexReport(s.ctx, req)
	s.ErrorContains(err, "ouch")
	s.Nil(r)

	// When get index report returns not found.
	s.indexerMock.
		EXPECT().
		GetIndexReport(gomock.Any(), gomock.Eq(hashID)).
		Return(nil, false, nil)
	r, err = s.service.GetIndexReport(s.ctx, req)
	s.ErrorContains(err, "not found")
	s.Nil(r)

	// When get index report returns an index report.
	s.indexerMock.
		EXPECT().
		GetIndexReport(gomock.Any(), gomock.Eq(hashID)).
		Return(&claircore.IndexReport{State: "sample state"}, true, nil)
	r, err = s.service.GetIndexReport(s.ctx, req)
	s.NoError(err)
	s.Equal(&v4.IndexReport{HashId: hashID, State: "sample state", Contents: &v4.Contents{}}, r)
}
