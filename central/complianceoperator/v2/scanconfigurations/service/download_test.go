package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	blobMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDownloadReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := sac.WithAllAccess(context.Background())

	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	blobDS := blobMocks.NewMockDatastore(ctrl)

	user := &storage.SlimUser{}
	user.SetId("user-1")
	user.SetName("user-1")

	handler := &downloadHandler{
		snapshotDataStore: snapshotDS,
		blobStore:         blobDS,
	}

	t.Run("Method not allowed", func(t *testing.T) {
		req, _ := http.NewRequest("POST", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", "snapshot-id"), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
	})

	t.Run("Empty ID", func(t *testing.T) {
		emptyID := ""

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", emptyID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("User not present in context", func(t *testing.T) {
		snapshotID := "snapshot-1"

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusForbidden, res.StatusCode)
	})

	t.Run("Snapshot Store error", func(t *testing.T) {
		snapshotID := "snapshot-1"

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(nil, false, errors.New("some error"))

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("Snapshot ID not found", func(t *testing.T) {
		snapshotID := "snapshot-1"

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(nil, false, nil)

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("Snapshot User differs from the User in the Context", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)
		ctxUser := &storage.SlimUser{}
		ctxUser.SetId("user-2")
		ctxUser.SetName("user-2")

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, ctxUser))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusForbidden, res.StatusCode)
	})

	t.Run("Snapshot is not a download", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)
		snapshot.GetReportStatus().SetReportNotificationMethod(storage.ComplianceOperatorReportStatus_EMAIL)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("Snapshot is waiting", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)
		snapshot.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_WAITING)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
	})

	t.Run("Snapshot is preparing", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)
		snapshot.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_PREPARING)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
	})

	t.Run("Snapshot failed", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)
		snapshot.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_FAILURE)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("Blob Store error", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		blobDS.EXPECT().Get(gomock.Any(), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID), gomock.Any()).Return(nil, false, errors.New("some error"))

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	})

	t.Run("Blob not found", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		blobDS.EXPECT().Get(gomock.Any(), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID), gomock.Any()).Return(nil, false, nil)

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("Snapshot Store upsert failure", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		blobDS.EXPECT().Get(gomock.Any(), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID), gomock.Any()).Return(nil, true, nil)

		clone := snapshot.CloneVT()
		clone.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_DELIVERED)
		snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), snapshot).DoAndReturn(func(_ any, s *storage.ComplianceOperatorReportSnapshotV2) error {
			require.Equal(t, clone.GetReportStatus().GetRunState(), s.GetReportStatus().GetRunState())
			return errors.New("some error")
		})

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("Snapshot Store upsert success", func(t *testing.T) {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, user)

		snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		blobDS.EXPECT().Get(gomock.Any(), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID), gomock.Any()).Return(nil, true, nil)
		clone := snapshot.CloneVT()
		clone.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_DELIVERED)
		snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), snapshot).DoAndReturn(func(_ any, s *storage.ComplianceOperatorReportSnapshotV2) error {
			require.Equal(t, clone.GetReportStatus().GetRunState(), s.GetReportStatus().GetRunState())
			return nil
		})

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://example.com/v2/compliance/scan/configurations/reports/download?id=%s", snapshotID), nil)
		req = req.WithContext(getContextForUser(t, ctrl, ctx, user))
		w := httptest.NewRecorder()
		handler.handle(w, req)
		res := w.Result()

		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}

func getSnapshot(id string, user *storage.SlimUser) *storage.ComplianceOperatorReportSnapshotV2 {
	cors := &storage.ComplianceOperatorReportStatus{}
	cors.SetReportNotificationMethod(storage.ComplianceOperatorReportStatus_DOWNLOAD)
	cors.SetRunState(storage.ComplianceOperatorReportStatus_GENERATED)
	corsv2 := &storage.ComplianceOperatorReportSnapshotV2{}
	corsv2.SetReportId(id)
	corsv2.SetName(id)
	corsv2.SetScanConfigurationId(fmt.Sprintf("scan-config-%s", id))
	corsv2.SetUser(user)
	corsv2.SetReportStatus(cors)
	return corsv2
}
