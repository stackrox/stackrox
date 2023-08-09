package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	blobDSMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type handlerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                     context.Context
	reportSnapshotDataStore *reportSnapshotDSMocks.MockDataStore
	blobStore               *blobDSMocks.MockDatastore
	handler                 *downloadHandler
}

func TestReportHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

func (s *handlerTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
	s.reportSnapshotDataStore = reportSnapshotDSMocks.NewMockDataStore(s.mockCtrl)
	s.blobStore = blobDSMocks.NewMockDatastore(s.mockCtrl)
	s.handler = &downloadHandler{
		snapshotStore: s.reportSnapshotDataStore,
		blobStore:     s.blobStore,
	}
}

func (s *handlerTestSuite) TestDownloadReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.ReportId = uuid.NewV4().String()
	reportSnapshot.ReportConfigurationId = uuid.NewV4().String()
	reportSnapshot.ReportStatus.RunState = storage.ReportStatus_SUCCESS
	reportSnapshot.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
	user := reportSnapshot.GetRequester()
	userContext := s.getContextForUser(user)
	blob, blobData := fixtures.GetBlobWithData()
	blobName := common.GetReportBlobPath(reportSnapshot.GetReportConfigurationId(), reportSnapshot.GetReportId())

	testCases := []struct {
		desc         string
		id           string
		ctx          context.Context
		mockGen      func()
		expectedCode int
		statusCode   int
	}{
		{
			desc:       "Empty Report ID",
			id:         "",
			ctx:        userContext,
			statusCode: http.StatusBadRequest,
		},
		{
			desc:       "User info not present in context",
			id:         reportSnapshot.GetReportId(),
			ctx:        s.ctx,
			mockGen:    func() {},
			statusCode: http.StatusForbidden,
		},
		{
			desc: "Report ID not found",
			id:   reportSnapshot.GetReportId(),
			ctx:  userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			statusCode: http.StatusNotFound,
		},
		{
			desc: "Download requester id and report requester user id mismatch",
			id:   reportSnapshot.GetReportId(),
			ctx:  userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.Requester = &storage.SlimUser{
					Id:   reportSnapshot.Requester.Id + "-1",
					Name: reportSnapshot.Requester.Name + "-1",
				}
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			statusCode: http.StatusForbidden,
		},
		{
			desc: "Report was not generated",
			id:   reportSnapshot.GetReportId(),
			ctx:  userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_EMAIL
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			statusCode: http.StatusBadRequest,
		},
		{
			desc: "Report is not ready yet",
			id:   reportSnapshot.GetReportId(),
			ctx:  userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_PREPARING
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			statusCode: http.StatusServiceUnavailable,
		},
		{
			desc: "Blob get error",
			id:   reportSnapshot.GetReportId(),
			ctx:  userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.blobStore.EXPECT().Get(gomock.Any(), blobName, gomock.Any()).Times(1).Return(nil, false, errors.New(""))
			},
			statusCode: http.StatusInternalServerError,
		},
		{
			desc: "Blob does not exist",
			id:   reportSnapshot.GetReportId(),
			ctx:  userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.blobStore.EXPECT().Get(gomock.Any(), blobName, gomock.Any()).Times(1).Return(nil, false, nil)
			},
			statusCode: http.StatusNotFound,
		},
		{
			desc: "Report downloaded",
			id:   reportSnapshot.GetReportId(),
			ctx:  userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.blobStore.EXPECT().Get(gomock.Any(), blobName, gomock.Any()).Times(1).DoAndReturn(
					func(_ context.Context, _ string, writer io.Writer) (*storage.Blob, bool, error) {
						c, err := writer.Write(blobData.Bytes())
						s.NoError(err)
						s.Equal(c, blobData.Len())
						return blob, true, nil
					})
			},
			statusCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			if tc.mockGen != nil {
				tc.mockGen()
			}
			s.downloadAndVerify(tc.ctx, tc.id, tc.statusCode, blobData.Bytes())
		})
	}
}

func (s *handlerTestSuite) downloadAndVerify(ctx context.Context, id string, code int, expectData []byte) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/api/reports/jobs/download/?id=%s", id), nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	s.handler.handle(w, req)
	result := w.Result()

	s.Equal(code, result.StatusCode)
	if result.StatusCode == http.StatusOK {
		data, err := io.ReadAll(result.Body)
		s.NoError(err)
		s.Equal(expectData, data)
	}
}

func (s *handlerTestSuite) getContextForUser(user *storage.SlimUser) context.Context {
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()
	return authn.ContextWithIdentity(s.ctx, mockID, s.T())
}
