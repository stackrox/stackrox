package reportgenerator

import (
	"context"
	"testing"

	snapshotMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestReportCancellationStatus(t *testing.T) {
	for name, cause := range map[string]error{"shutdown": ErrSchedulerStopped, "user cancellation": ErrUserCancelled} {
		t.Run(name, func(t *testing.T) {
			store := snapshotMocks.NewMockDataStore(gomock.NewController(t))
			snap := &storage.ReportSnapshot{ReportId: "interrupted", ReportStatus: &storage.ReportStatus{RunState: storage.ReportStatus_PREPARING}}
			if cause == ErrUserCancelled {
				store.EXPECT().UpdateReportSnapshot(gomock.Any(), snap).Return(nil)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(cause)
			LogAndUpsertError(ctx, context.Background(), store, ctx.Err(), &ReportRequest{ReportSnapshot: snap})
			if cause == ErrSchedulerStopped {
				assert.Equal(t, storage.ReportStatus_PREPARING, snap.GetReportStatus().GetRunState())
				assert.Nil(t, snap.GetReportStatus().GetCompletedAt())
			} else {
				assert.Equal(t, storage.ReportStatus_FAILURE, snap.GetReportStatus().GetRunState())
				assert.Equal(t, ErrUserCancelled.Error(), snap.GetReportStatus().GetErrorMsg())
			}
		})
	}
}

func TestShutdownAfterGeneration(t *testing.T) {
	for name, method := range map[string]storage.ReportStatus_NotificationMethod{"email": storage.ReportStatus_EMAIL, "download": storage.ReportStatus_DOWNLOAD} {
		t.Run(name, func(t *testing.T) {
			store := snapshotMocks.NewMockDataStore(gomock.NewController(t))
			snap := &storage.ReportSnapshot{ReportStatus: &storage.ReportStatus{RunState: storage.ReportStatus_GENERATED, ReportNotificationMethod: method, CompletedAt: protocompat.TimestampNow()}}
			if method == storage.ReportStatus_EMAIL {
				store.EXPECT().UpdateReportSnapshot(gomock.Any(), snap).Return(nil)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(ErrSchedulerStopped)
			LogAndUpsertError(ctx, context.Background(), store, ctx.Err(), &ReportRequest{ReportSnapshot: snap})
			if method == storage.ReportStatus_EMAIL {
				assert.Equal(t, storage.ReportStatus_WAITING, snap.GetReportStatus().GetRunState())
				assert.Nil(t, snap.GetReportStatus().GetCompletedAt())
			} else {
				assert.Equal(t, storage.ReportStatus_GENERATED, snap.GetReportStatus().GetRunState())
			}
		})
	}
}
