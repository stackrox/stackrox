package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store postgres.Store
}

func (d *datastoreImpl) GetSnapshot(ctx context.Context, id string) (*storage.ComplianceOperatorReportSnapshotV2, bool, error) {
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) SearchSnapshots(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
	return d.store.GetByQuery(ctx, query)
}

func (d *datastoreImpl) UpsertSnapshot(ctx context.Context, result *storage.ComplianceOperatorReportSnapshotV2) error {
	return d.store.Upsert(ctx, result)
}

func (d *datastoreImpl) DeleteSnapshot(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// DeleteOrphanedReportSnapshots deletes all the snapshots that are not in a final state
// This is only called on startup to make sure we do not have orphaned reports
func DeleteOrphanedReportSnapshots(ctx context.Context, ds DataStore) error {
	errList := errorhelpers.NewErrorList("delete orphaned reports")
	searchQueryForEmails := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorReportState,
			storage.ComplianceOperatorReportStatus_WAITING.String(),
			storage.ComplianceOperatorReportStatus_PREPARING.String(),
			storage.ComplianceOperatorReportStatus_GENERATED.String(),
		).AddExactMatches(search.ComplianceOperatorReportNotificationMethod,
		storage.ComplianceOperatorReportStatus_EMAIL.String()).ProtoQuery()
	if err := deleteOrphanedSnapshots(ctx, ds, searchQueryForEmails); err != nil {
		errList.AddErrors(err)
	}
	// For report of type Download we do not purge if the state is GENERATED
	// as they can still be downloaded.
	searchQueryForDownloads := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorReportState,
			storage.ComplianceOperatorReportStatus_WAITING.String(),
			storage.ComplianceOperatorReportStatus_PREPARING.String(),
		).AddExactMatches(search.ComplianceOperatorReportNotificationMethod,
		storage.ComplianceOperatorReportStatus_DOWNLOAD.String()).ProtoQuery()
	if err := deleteOrphanedSnapshots(ctx, ds, searchQueryForDownloads); err != nil {
		errList.AddErrors(err)
	}
	return errList.ToError()
}

func deleteOrphanedSnapshots(ctx context.Context, ds DataStore, query *v1.Query) error {
	orphanSnapshots, err := ds.SearchSnapshots(ctx, query)
	if err != nil {
		return errors.Wrap(err, "unable to search for orphan snapshots")

	}
	errList := errorhelpers.NewErrorList("search and delete orphaned reports")
	for _, snapshot := range orphanSnapshots {
		if err := ds.DeleteSnapshot(ctx, snapshot.GetReportId()); err != nil {
			errList.AddErrors(errors.Wrapf(err, "unable to delete snapshot %s", snapshot.GetReportId()))
		}
	}
	return errList.ToError()
}

func (d *datastoreImpl) GetLastSnapshotFromScanConfig(ctx context.Context, scanConfigID string) (*storage.ComplianceOperatorReportSnapshotV2, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).ProtoQuery()
	snapshots, err := d.SearchSnapshots(ctx, query)
	if err != nil {
		return nil, err
	}
	var lastSnapshot *storage.ComplianceOperatorReportSnapshotV2
	for _, snapshot := range snapshots {
		if types.CompareTimestamps(snapshot.GetReportStatus().GetCompletedAt(), lastSnapshot.GetReportStatus().GetCompletedAt()) > 0 {
			lastSnapshot = snapshot
		}
	}
	return lastSnapshot, nil
}
