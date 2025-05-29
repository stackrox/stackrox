package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/report/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
