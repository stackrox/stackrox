package utils

import (
	"context"

	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

// UpdateSnapshotOnError updates the state of a given snapshot to FAILURE
func UpdateSnapshotOnError(ctx context.Context, snapshot *storage.ComplianceOperatorReportSnapshotV2, err error, store snapshotDS.DataStore) error {
	if snapshot == nil {
		return nil
	}
	snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_FAILURE
	snapshot.GetReportStatus().ErrorMsg = err.Error()
	snapshot.GetReportStatus().CompletedAt = protocompat.TimestampNow()
	if dbErr := store.UpsertSnapshot(ctx, snapshot); dbErr != nil {
		return dbErr
	}
	return nil
}
